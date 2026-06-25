package views

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestScaffold(t *testing.T) {
	tests := []struct {
		name     string
		fw       Framework
		source   string
		contains string
		file     string
	}{
		{"stars", FrameworkQuarto, "duckdb", "duckdb.connect", "view.qmd"},
		{"stars", FrameworkQuarto, "parquet", "read_parquet", "view.qmd"},
		{"overview", FrameworkMarimo, "duckdb", "duckdb.connect", "app.py"},
		{"overview", FrameworkMarimo, "parquet", "read_parquet", "app.py"},
		{"analysis", FrameworkJupyter, "duckdb", "duckdb.connect", "view.ipynb"},
		{"analysis", FrameworkJupyter, "parquet", "read_parquet", "view.ipynb"},
		{"trend", FrameworkR, "duckdb", "dbConnect(duckdb()", "view.R"},
		{"trend", FrameworkR, "parquet", "read_parquet", "view.R"},
		{"badge", FrameworkSQL, "duckdb", ".duckdb", "view.sql"},
		{"badge", FrameworkSQL, "parquet", "read_parquet", "view.sql"},
	}

	for _, tt := range tests {
		t.Run(string(tt.fw)+"_"+tt.source, func(t *testing.T) {
			viewsDir := t.TempDir()

			dir, err := Scaffold(ScaffoldOptions{
				ViewsDir:  viewsDir,
				Name:      tt.name,
				Framework: tt.fw,
				Source:    tt.source,
				DBPath:    "../../data/velocirepo.duckdb",
				DataDir:   "../_data",
			})
			if err != nil {
				t.Fatal(err)
			}

			viewFile := filepath.Join(dir, tt.file)
			content, err := os.ReadFile(viewFile)
			if err != nil {
				t.Fatalf("view file not created: %v", err)
			}

			if !strings.Contains(string(content), tt.contains) {
				t.Errorf("view file does not contain %q:\n%s", tt.contains, content)
			}

			renderSh := filepath.Join(dir, "render.sh")
			info, err := os.Stat(renderSh)
			if err != nil {
				t.Fatal("render.sh not created")
			}
			if info.Mode().Perm()&0100 == 0 {
				t.Error("render.sh is not executable")
			}
		})
	}
}

func TestScaffoldPyproject(t *testing.T) {
	for _, fw := range []Framework{FrameworkQuarto, FrameworkJupyter, FrameworkMarimo} {
		t.Run(string(fw), func(t *testing.T) {
			viewsDir := t.TempDir()

			dir, err := Scaffold(ScaffoldOptions{
				ViewsDir:  viewsDir,
				Name:      "test",
				Framework: fw,
				Source:    "duckdb",
				DBPath:    "../../data/velocirepo.duckdb",
			})
			if err != nil {
				t.Fatal(err)
			}

			pyproject := filepath.Join(dir, "pyproject.toml")
			if _, err := os.Stat(pyproject); err != nil {
				t.Error("pyproject.toml not created for Python framework")
			}
		})
	}
}

func TestScaffoldNoUV(t *testing.T) {
	viewsDir := t.TempDir()

	dir, err := Scaffold(ScaffoldOptions{
		ViewsDir:  viewsDir,
		Name:      "test",
		Framework: FrameworkQuarto,
		Source:    "duckdb",
		DBPath:    "../../data/velocirepo.duckdb",
		NoUV:      true,
	})
	if err != nil {
		t.Fatal(err)
	}

	pyproject := filepath.Join(dir, "pyproject.toml")
	if _, err := os.Stat(pyproject); !os.IsNotExist(err) {
		t.Error("pyproject.toml should not be created with --no-uv")
	}
}

func TestScaffoldRenv(t *testing.T) {
	viewsDir := t.TempDir()

	dir, err := Scaffold(ScaffoldOptions{
		ViewsDir:  viewsDir,
		Name:      "r-view",
		Framework: FrameworkR,
		Source:    "duckdb",
		DBPath:    "../../data/velocirepo.duckdb",
		Renv:      true,
	})
	if err != nil {
		t.Fatal(err)
	}

	rprofile := filepath.Join(dir, ".Rprofile")
	if _, err := os.Stat(rprofile); err != nil {
		t.Error(".Rprofile not created")
	}

	renvSettings := filepath.Join(dir, "renv", "settings.json")
	if _, err := os.Stat(renvSettings); err != nil {
		t.Error("renv/settings.json not created")
	}

	renderSh := filepath.Join(dir, "render.sh")
	content, _ := os.ReadFile(renderSh)
	if !strings.Contains(string(content), "renv::restore") {
		t.Error("render.sh should contain renv::restore")
	}
}

func TestScaffoldRNoRenv(t *testing.T) {
	viewsDir := t.TempDir()

	dir, err := Scaffold(ScaffoldOptions{
		ViewsDir:  viewsDir,
		Name:      "r-plain",
		Framework: FrameworkR,
		Source:    "duckdb",
		DBPath:    "../../data/velocirepo.duckdb",
	})
	if err != nil {
		t.Fatal(err)
	}

	rprofile := filepath.Join(dir, ".Rprofile")
	if _, err := os.Stat(rprofile); !os.IsNotExist(err) {
		t.Error(".Rprofile should not exist without --renv")
	}

	// Should NOT have pyproject.toml for R
	pyproject := filepath.Join(dir, "pyproject.toml")
	if _, err := os.Stat(pyproject); !os.IsNotExist(err) {
		t.Error("pyproject.toml should not be created for R framework")
	}
}

func TestScaffoldSubdirectory(t *testing.T) {
	viewsDir := t.TempDir()

	dir, err := Scaffold(ScaffoldOptions{
		ViewsDir:  viewsDir,
		Name:      "weekly/stars",
		Framework: FrameworkQuarto,
		Source:    "duckdb",
		DBPath:    "../../../data/velocirepo.duckdb",
	})
	if err != nil {
		t.Fatal(err)
	}

	expected := filepath.Join(viewsDir, "weekly", "stars")
	if dir != expected {
		t.Errorf("dir = %q, want %q", dir, expected)
	}

	if _, err := os.Stat(filepath.Join(dir, "render.sh")); err != nil {
		t.Error("render.sh not created in subdirectory")
	}
}

func TestScaffoldAlreadyExists(t *testing.T) {
	viewsDir := t.TempDir()

	os.MkdirAll(filepath.Join(viewsDir, "existing"), 0755)

	_, err := Scaffold(ScaffoldOptions{
		ViewsDir:  viewsDir,
		Name:      "existing",
		Framework: FrameworkQuarto,
		Source:    "duckdb",
		DBPath:    "../../data/velocirepo.duckdb",
	})
	if err == nil {
		t.Error("expected error for existing directory")
	}
}
