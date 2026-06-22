package views

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestScaffold(t *testing.T) {
	tests := []struct {
		name      string
		fw        Framework
		source    string
		contains  string
	}{
		{"stars", FrameworkQuarto, "parquet", "read_parquet"},
		{"downloads", FrameworkQuarto, "jsonl", "read_json_auto"},
		{"overview", FrameworkMarimo, "parquet", "read_parquet"},
		{"overview", FrameworkMarimo, "jsonl", "read_json_auto"},
		{"analysis", FrameworkJupyter, "parquet", "read_parquet"},
		{"analysis", FrameworkJupyter, "jsonl", "read_json_auto"},
		{"trend", FrameworkR, "parquet", "read_parquet"},
		{"trend", FrameworkR, "jsonl", "read_json_auto"},
		{"badge", FrameworkSQL, "parquet", "read_parquet"},
		{"badge", FrameworkSQL, "jsonl", "read_json_auto"},
	}

	for _, tt := range tests {
		t.Run(string(tt.fw)+"_"+tt.source, func(t *testing.T) {
			dir := t.TempDir()
			dataDir := filepath.Join(dir, "data")

			path, err := Scaffold(dir, tt.name, tt.fw, tt.source, dataDir)
			if err != nil {
				t.Fatal(err)
			}

			content, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}

			if !strings.Contains(string(content), tt.contains) {
				t.Errorf("scaffolded file does not contain %q:\n%s", tt.contains, content)
			}

			ext := ExtForFramework(tt.fw)
			expectedPath := filepath.Join(dir, tt.name+ext)
			if path != expectedPath {
				t.Errorf("path = %q, want %q", path, expectedPath)
			}
		})
	}
}

func TestScaffoldCreatesGitignore(t *testing.T) {
	dir := t.TempDir()
	dataDir := filepath.Join(dir, "data")

	_, err := Scaffold(dir, "test", FrameworkSQL, "parquet", dataDir)
	if err != nil {
		t.Fatal(err)
	}

	gitignore := filepath.Join(dir, ".gitignore")
	content, err := os.ReadFile(gitignore)
	if err != nil {
		t.Fatal("expected .gitignore to be created")
	}

	if !strings.Contains(string(content), "_data/") {
		t.Error(".gitignore missing _data/")
	}
	if !strings.Contains(string(content), "_output/") {
		t.Error(".gitignore missing _output/")
	}
}

func TestScaffoldSubdirectory(t *testing.T) {
	dir := t.TempDir()
	dataDir := filepath.Join(dir, "data")

	path, err := Scaffold(dir, "weekly/stars", FrameworkQuarto, "parquet", dataDir)
	if err != nil {
		t.Fatal(err)
	}

	expected := filepath.Join(dir, "weekly", "stars.qmd")
	if path != expected {
		t.Errorf("path = %q, want %q", path, expected)
	}

	if _, err := os.Stat(path); err != nil {
		t.Errorf("scaffolded file does not exist at %s", path)
	}
}
