package views

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jeroenjanssens/velocirepo/internal/config"
)

func TestParseFramework(t *testing.T) {
	tests := []struct {
		input string
		want  Framework
		err   bool
	}{
		{"quarto", FrameworkQuarto, false},
		{"Quarto", FrameworkQuarto, false},
		{"jupyter", FrameworkJupyter, false},
		{"marimo", FrameworkMarimo, false},
		{"r", FrameworkR, false},
		{"R", FrameworkR, false},
		{"sql", FrameworkSQL, false},
		{"invalid", "", true},
	}

	for _, tt := range tests {
		got, err := ParseFramework(tt.input)
		if tt.err && err == nil {
			t.Errorf("ParseFramework(%q) expected error", tt.input)
		}
		if !tt.err && err != nil {
			t.Errorf("ParseFramework(%q) unexpected error: %v", tt.input, err)
		}
		if got != tt.want {
			t.Errorf("ParseFramework(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestExtForFramework(t *testing.T) {
	tests := []struct {
		fw   Framework
		want string
	}{
		{FrameworkQuarto, ".qmd"},
		{FrameworkJupyter, ".ipynb"},
		{FrameworkMarimo, ".py"},
		{FrameworkR, ".R"},
		{FrameworkSQL, ".sql"},
	}

	for _, tt := range tests {
		got := ExtForFramework(tt.fw)
		if got != tt.want {
			t.Errorf("ExtForFramework(%q) = %q, want %q", tt.fw, got, tt.want)
		}
	}
}

func TestDiscover(t *testing.T) {
	dir := t.TempDir()

	os.MkdirAll(filepath.Join(dir, "weekly"), 0755)
	os.WriteFile(filepath.Join(dir, "overview.py"), []byte("# marimo"), 0644)
	os.WriteFile(filepath.Join(dir, "weekly", "stars.qmd"), []byte("---\ntitle: stars\n---"), 0644)
	os.WriteFile(filepath.Join(dir, "report.sql"), []byte("SELECT 1"), 0644)
	os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("not a view"), 0644)

	// Create _output and _data dirs that should be skipped
	os.MkdirAll(filepath.Join(dir, "_output"), 0755)
	os.WriteFile(filepath.Join(dir, "_output", "old.html"), []byte(""), 0644)
	os.MkdirAll(filepath.Join(dir, "_data"), 0755)
	os.WriteFile(filepath.Join(dir, "_data", "metrics.parquet"), []byte(""), 0644)

	views, err := Discover(dir, nil, "parquet")
	if err != nil {
		t.Fatal(err)
	}

	if len(views) != 3 {
		t.Fatalf("expected 3 views, got %d", len(views))
	}

	names := map[string]bool{}
	for _, v := range views {
		names[v.Name] = true
		if v.Source != "parquet" {
			t.Errorf("view %q has source %q, want parquet", v.Name, v.Source)
		}
	}

	for _, want := range []string{"overview", "weekly/stars", "report"} {
		if !names[want] {
			t.Errorf("expected view %q not found", want)
		}
	}
}

func TestDiscoverWithItems(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "overview.py"), []byte("# marimo"), 0644)

	items := []config.ViewItem{
		{Path: "overview.py", Source: "jsonl", Venv: "/path/to/venv"},
	}

	views, err := Discover(dir, items, "parquet")
	if err != nil {
		t.Fatal(err)
	}

	if len(views) != 1 {
		t.Fatalf("expected 1 view, got %d", len(views))
	}

	v := views[0]
	if v.Source != "jsonl" {
		t.Errorf("source = %q, want jsonl", v.Source)
	}
	if v.Venv != "/path/to/venv" {
		t.Errorf("venv = %q, want /path/to/venv", v.Venv)
	}
}

func TestFindView(t *testing.T) {
	views := []View{
		{Name: "overview"},
		{Name: "weekly/stars"},
	}

	v, found := FindView(views, "weekly/stars")
	if !found {
		t.Fatal("expected to find weekly/stars")
	}
	if v.Name != "weekly/stars" {
		t.Errorf("name = %q, want weekly/stars", v.Name)
	}

	_, found = FindView(views, "nonexistent")
	if found {
		t.Error("expected not to find nonexistent")
	}
}

func TestAnyUsesParquet(t *testing.T) {
	if AnyUsesParquet([]View{{Source: "jsonl"}}) {
		t.Error("expected false for all-jsonl views")
	}
	if !AnyUsesParquet([]View{{Source: "jsonl"}, {Source: "parquet"}}) {
		t.Error("expected true when one view uses parquet")
	}
}
