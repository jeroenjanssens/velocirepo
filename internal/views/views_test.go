package views

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseFramework(t *testing.T) {
	tests := []struct {
		input string
		want  Framework
		err   bool
	}{
		{"quarto-python", FrameworkQuartoPython, false},
		{"quarto-r", FrameworkQuartoR, false},
		{"jupyter", FrameworkJupyter, false},
		{"marimo", FrameworkMarimo, false},
		{"r", FrameworkR, false},
		{"sql", FrameworkSQL, false},
		{"quarto", "", true},
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

func TestDiscover(t *testing.T) {
	dir := t.TempDir()

	// Create view directories with render.sh
	_ = os.MkdirAll(filepath.Join(dir, "overview"), 0755)
	_ = os.WriteFile(filepath.Join(dir, "overview", "render.sh"), []byte("#!/bin/bash\n"), 0755)

	_ = os.MkdirAll(filepath.Join(dir, "weekly", "stars"), 0755)
	_ = os.WriteFile(filepath.Join(dir, "weekly", "stars", "render.sh"), []byte("#!/bin/bash\n"), 0755)

	_ = os.MkdirAll(filepath.Join(dir, "weekly", "forks"), 0755)
	_ = os.WriteFile(filepath.Join(dir, "weekly", "forks", "render.sh"), []byte("#!/bin/bash\n"), 0755)

	// Directories without render.sh should be skipped
	_ = os.MkdirAll(filepath.Join(dir, "draft"), 0755)
	_ = os.WriteFile(filepath.Join(dir, "draft", "notes.txt"), []byte("not a view"), 0644)

	// _output and _data dirs should be skipped
	_ = os.MkdirAll(filepath.Join(dir, "_output"), 0755)
	_ = os.WriteFile(filepath.Join(dir, "_output", "render.sh"), []byte("#!/bin/bash\n"), 0755)
	_ = os.MkdirAll(filepath.Join(dir, "_data"), 0755)

	// Hidden dirs should be skipped
	_ = os.MkdirAll(filepath.Join(dir, ".hidden"), 0755)
	_ = os.WriteFile(filepath.Join(dir, ".hidden", "render.sh"), []byte("#!/bin/bash\n"), 0755)

	views, err := Discover(dir)
	if err != nil {
		t.Fatal(err)
	}

	if len(views) != 3 {
		names := make([]string, len(views))
		for i, v := range views {
			names[i] = v.Name
		}
		t.Fatalf("expected 3 views, got %d: %v", len(views), names)
	}

	nameSet := map[string]bool{}
	for _, v := range views {
		nameSet[v.Name] = true
	}

	for _, want := range []string{"overview", "weekly/stars", "weekly/forks"} {
		if !nameSet[want] {
			t.Errorf("expected view %q not found", want)
		}
	}
}

func TestDiscoverEmpty(t *testing.T) {
	dir := t.TempDir()
	views, err := Discover(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(views) != 0 {
		t.Errorf("expected 0 views, got %d", len(views))
	}
}

func TestDiscoverNonexistent(t *testing.T) {
	views, err := Discover("/nonexistent/path")
	if err != nil {
		t.Fatal(err)
	}
	if views != nil {
		t.Errorf("expected nil views for nonexistent dir, got %v", views)
	}
}

func TestFindView(t *testing.T) {
	views := []View{
		{Name: "overview", Dir: "/views/overview"},
		{Name: "weekly/stars", Dir: "/views/weekly/stars"},
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

func TestFindViews(t *testing.T) {
	allViews := []View{
		{Name: "overview"},
		{Name: "weekly/stars"},
		{Name: "weekly/forks"},
		{Name: "monthly/summary"},
	}

	got := FindViews(allViews, "overview")
	if len(got) != 1 || got[0].Name != "overview" {
		t.Errorf("exact match: got %v, want [overview]", got)
	}

	got = FindViews(allViews, "weekly")
	if len(got) != 2 {
		t.Fatalf("directory match: got %d views, want 2", len(got))
	}

	got = FindViews(allViews, "weekly/")
	if len(got) != 2 {
		t.Fatalf("directory match with slash: got %d views, want 2", len(got))
	}

	got = FindViews(allViews, "nonexistent")
	if len(got) != 0 {
		t.Errorf("no match: got %v, want []", got)
	}
}
