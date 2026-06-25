package views

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

//go:embed templates/*
var templateFS embed.FS

type Framework string

const (
	FrameworkQuarto  Framework = "quarto"
	FrameworkJupyter Framework = "jupyter"
	FrameworkMarimo  Framework = "marimo"
	FrameworkR       Framework = "r"
	FrameworkSQL     Framework = "sql"
)

func ParseFramework(s string) (Framework, error) {
	switch strings.ToLower(s) {
	case "quarto":
		return FrameworkQuarto, nil
	case "jupyter":
		return FrameworkJupyter, nil
	case "marimo":
		return FrameworkMarimo, nil
	case "r":
		return FrameworkR, nil
	case "sql":
		return FrameworkSQL, nil
	default:
		return "", fmt.Errorf("unknown framework %q (available: quarto, jupyter, marimo, r, sql)", s)
	}
}

type ScaffoldOptions struct {
	ViewsDir string
	Name     string
	Framework Framework
	Source   string
	DBPath   string
	DataDir  string
	NoUV     bool
	Renv     bool
}

type scaffoldData struct {
	ViewName string
	DBPath   string
	DataDir  string
}

func Scaffold(opts ScaffoldOptions) (string, error) {
	dir := filepath.Join(opts.ViewsDir, opts.Name)
	if _, err := os.Stat(dir); err == nil {
		return "", fmt.Errorf("view directory %q already exists", dir)
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("create view directory: %w", err)
	}

	data := scaffoldData{
		ViewName: filepath.Base(opts.Name),
		DBPath:   opts.DBPath,
		DataDir:  opts.DataDir,
	}

	renderTmpl := renderShTemplate(opts.Framework, opts.Renv)
	if err := writeTemplate(dir, "render.sh", renderTmpl, data, 0755); err != nil {
		return "", err
	}

	viewFile, viewTmpl := viewFileTemplate(opts.Framework, opts.Source)
	if err := writeTemplate(dir, viewFile, viewTmpl, data, 0644); err != nil {
		return "", err
	}

	if serveTmpl := serveShTemplate(opts.Framework); serveTmpl != "" {
		if err := writeTemplate(dir, "serve.sh", serveTmpl, data, 0755); err != nil {
			return "", err
		}
	}

	if needsPyproject(opts.Framework) && !opts.NoUV {
		pyTmpl := fmt.Sprintf("templates/%s/pyproject.toml.tmpl", opts.Framework)
		if err := writeTemplate(dir, "pyproject.toml", pyTmpl, data, 0644); err != nil {
			return "", err
		}
	}

	if opts.Framework == FrameworkR && opts.Renv {
		if err := scaffoldRenv(dir); err != nil {
			return "", err
		}
	}

	return dir, nil
}

func renderShTemplate(fw Framework, renv bool) string {
	if fw == FrameworkR && renv {
		return "templates/r/render.renv.sh.tmpl"
	}
	return fmt.Sprintf("templates/%s/render.sh.tmpl", fw)
}

func serveShTemplate(fw Framework) string {
	switch fw {
	case FrameworkQuarto, FrameworkJupyter, FrameworkMarimo:
		return fmt.Sprintf("templates/%s/serve.sh.tmpl", fw)
	default:
		return ""
	}
}

func viewFileTemplate(fw Framework, source string) (filename, tmplPath string) {
	switch fw {
	case FrameworkQuarto:
		return "view.qmd", fmt.Sprintf("templates/quarto/view.qmd.%s.tmpl", source)
	case FrameworkJupyter:
		return "view.ipynb", fmt.Sprintf("templates/jupyter/view.ipynb.%s.tmpl", source)
	case FrameworkMarimo:
		return "app.py", fmt.Sprintf("templates/marimo/app.py.%s.tmpl", source)
	case FrameworkR:
		return "view.R", fmt.Sprintf("templates/r/view.R.%s.tmpl", source)
	case FrameworkSQL:
		return "view.sql", fmt.Sprintf("templates/sql/view.sql.%s.tmpl", source)
	default:
		return "view.txt", ""
	}
}

func needsPyproject(fw Framework) bool {
	return fw == FrameworkQuarto || fw == FrameworkJupyter || fw == FrameworkMarimo
}

func writeTemplate(dir, filename, tmplPath string, data scaffoldData, perm os.FileMode) error {
	tmplData, err := templateFS.ReadFile(tmplPath)
	if err != nil {
		return fmt.Errorf("read template %s: %w", tmplPath, err)
	}

	tmpl, err := template.New(filename).Parse(string(tmplData))
	if err != nil {
		return fmt.Errorf("parse template %s: %w", tmplPath, err)
	}

	path := filepath.Join(dir, filename)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, perm)
	if err != nil {
		return fmt.Errorf("create %s: %w", filename, err)
	}
	defer f.Close()

	if err := tmpl.Execute(f, data); err != nil {
		return fmt.Errorf("execute template %s: %w", tmplPath, err)
	}
	return nil
}

func scaffoldRenv(dir string) error {
	rprofile := `source("renv/activate.R")` + "\n"
	if err := os.WriteFile(filepath.Join(dir, ".Rprofile"), []byte(rprofile), 0644); err != nil {
		return fmt.Errorf("create .Rprofile: %w", err)
	}
	renvDir := filepath.Join(dir, "renv")
	if err := os.MkdirAll(renvDir, 0755); err != nil {
		return fmt.Errorf("create renv dir: %w", err)
	}
	settings := `{"external.libraries":[],"ignored.packages":[],"snapshot.type":"implicit","use.cache":true}` + "\n"
	if err := os.WriteFile(filepath.Join(renvDir, "settings.json"), []byte(settings), 0644); err != nil {
		return fmt.Errorf("create renv settings: %w", err)
	}
	return nil
}
