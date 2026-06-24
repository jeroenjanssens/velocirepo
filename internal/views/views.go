package views

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jeroenjanssens/velocirepo/internal/config"
)

type Framework string

const (
	FrameworkQuarto  Framework = "quarto"
	FrameworkJupyter Framework = "jupyter"
	FrameworkMarimo  Framework = "marimo"
	FrameworkR       Framework = "r"
	FrameworkSQL     Framework = "sql"
)

var frameworkExtensions = map[Framework]string{
	FrameworkQuarto:  ".qmd",
	FrameworkJupyter: ".ipynb",
	FrameworkMarimo:  ".py",
	FrameworkR:       ".R",
	FrameworkSQL:     ".sql",
}

var extensionFrameworks = map[string]Framework{
	".qmd":   FrameworkQuarto,
	".ipynb":  FrameworkJupyter,
	".py":    FrameworkMarimo,
	".R":     FrameworkR,
	".sql":   FrameworkSQL,
}

func ExtForFramework(fw Framework) string {
	return frameworkExtensions[fw]
}

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

type View struct {
	Name      string
	Path      string
	Framework Framework
	Source    string
	Output    string
	Venv      string
}

func Discover(viewsDir string, items []config.ViewItem, defaultSource string) ([]View, error) {
	if defaultSource == "" {
		defaultSource = "parquet"
	}

	itemMap := make(map[string]config.ViewItem)
	for _, item := range items {
		name := strings.TrimSuffix(item.Path, filepath.Ext(item.Path))
		itemMap[name] = item
	}

	var views []View
	err := filepath.Walk(viewsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			base := filepath.Base(path)
			if base == "_output" || base == "_data" {
				return filepath.SkipDir
			}
			return nil
		}

		ext := filepath.Ext(path)
		fw, ok := extensionFrameworks[ext]
		if !ok {
			return nil
		}

		rel, _ := filepath.Rel(viewsDir, path)
		name := strings.TrimSuffix(rel, ext)

		view := View{
			Name:      name,
			Path:      path,
			Framework: fw,
			Source:    defaultSource,
		}

		if item, exists := itemMap[name]; exists {
			if item.Source != "" {
				view.Source = item.Source
			}
			if item.Output != "" {
				view.Output = item.Output
			}
			if item.Venv != "" {
				view.Venv = item.Venv
			}
			delete(itemMap, name)
		}

		if view.Output == "" {
			view.Output = defaultOutput(viewsDir, name, fw)
		}

		views = append(views, view)
		return nil
	})
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	return views, nil
}

func defaultOutput(viewsDir, name string, fw Framework) string {
	ext := ".html"
	if fw == FrameworkSQL {
		ext = ".json"
	}
	return filepath.Join(viewsDir, "_output", name+ext)
}

func FindView(views []View, name string) (View, bool) {
	for _, v := range views {
		if v.Name == name {
			return v, true
		}
	}
	return View{}, false
}

func FindViews(views []View, name string) []View {
	for _, v := range views {
		if v.Name == name {
			return []View{v}
		}
	}
	prefix := strings.TrimSuffix(name, "/") + "/"
	var matched []View
	for _, v := range views {
		if strings.HasPrefix(v.Name, prefix) {
			matched = append(matched, v)
		}
	}
	return matched
}

func AnyUsesParquet(views []View) bool {
	for _, v := range views {
		if v.Source == "parquet" {
			return true
		}
	}
	return false
}
