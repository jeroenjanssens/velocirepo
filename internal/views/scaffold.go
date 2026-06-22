package views

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"text/template"
)

//go:embed templates/*
var templateFS embed.FS

type ScaffoldData struct {
	ViewName     string
	DataDir      string
	ViewsDataDir string
}

func Scaffold(viewsDir, name string, fw Framework, source, dataDir string) (string, error) {
	ext := ExtForFramework(fw)
	if ext == "" {
		return "", fmt.Errorf("unknown framework %q", fw)
	}

	path := filepath.Join(viewsDir, name+ext)

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return "", fmt.Errorf("create directory: %w", err)
	}

	tmplName := fmt.Sprintf("templates/%s_%s%s.tmpl", fw, source, ext)
	tmplData, err := templateFS.ReadFile(tmplName)
	if err != nil {
		return "", fmt.Errorf("read template %s: %w", tmplName, err)
	}

	tmpl, err := template.New(name).Parse(string(tmplData))
	if err != nil {
		return "", fmt.Errorf("parse template: %w", err)
	}

	viewsDataDir := filepath.Join(viewsDir, "_data")

	data := ScaffoldData{
		ViewName:     name,
		DataDir:      dataDir,
		ViewsDataDir: viewsDataDir,
	}

	f, err := os.Create(path)
	if err != nil {
		return "", fmt.Errorf("create file: %w", err)
	}
	defer f.Close()

	if err := tmpl.Execute(f, data); err != nil {
		return "", fmt.Errorf("execute template: %w", err)
	}

	ensureGitignore(viewsDir)

	return path, nil
}

func ensureGitignore(viewsDir string) {
	gitignorePath := filepath.Join(viewsDir, ".gitignore")
	if _, err := os.Stat(gitignorePath); err == nil {
		return
	}
	os.WriteFile(gitignorePath, []byte("_data/\n_output/\n"), 0644)
}
