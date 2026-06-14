package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseGitHubURLSSH(t *testing.T) {
	owner, repo := parseGitHubURL("git@github.com:jeroenjanssens/velocirepo.git")
	if owner != "jeroenjanssens" {
		t.Errorf("owner = %q, want %q", owner, "jeroenjanssens")
	}
	if repo != "velocirepo" {
		t.Errorf("repo = %q, want %q", repo, "velocirepo")
	}
}

func TestParseGitHubURLSSHNoGit(t *testing.T) {
	owner, repo := parseGitHubURL("git@github.com:org/project")
	if owner != "org" {
		t.Errorf("owner = %q, want %q", owner, "org")
	}
	if repo != "project" {
		t.Errorf("repo = %q, want %q", repo, "project")
	}
}

func TestParseGitHubURLHTTPS(t *testing.T) {
	owner, repo := parseGitHubURL("https://github.com/posit-dev/great-tables.git")
	if owner != "posit-dev" {
		t.Errorf("owner = %q, want %q", owner, "posit-dev")
	}
	if repo != "great-tables" {
		t.Errorf("repo = %q, want %q", repo, "great-tables")
	}
}

func TestParseGitHubURLHTTPSNoGit(t *testing.T) {
	owner, repo := parseGitHubURL("https://github.com/org/repo")
	if owner != "org" {
		t.Errorf("owner = %q, want %q", owner, "org")
	}
	if repo != "repo" {
		t.Errorf("repo = %q, want %q", repo, "repo")
	}
}

func TestParseGitHubURLNonGitHub(t *testing.T) {
	owner, repo := parseGitHubURL("https://gitlab.com/org/repo.git")
	if owner != "" || repo != "" {
		t.Errorf("expected empty for non-GitHub URL, got %q/%q", owner, repo)
	}
}

func TestParseGitHubURLEmpty(t *testing.T) {
	owner, repo := parseGitHubURL("")
	if owner != "" || repo != "" {
		t.Errorf("expected empty for empty URL, got %q/%q", owner, repo)
	}
}

func TestDetectPyPI(t *testing.T) {
	dir := t.TempDir()
	content := `[project]
name = "great-tables"
version = "0.1.0"
`
	os.WriteFile(filepath.Join(dir, "pyproject.toml"), []byte(content), 0644)

	result := DetectPyPI(dir)
	if result != "great-tables" {
		t.Errorf("DetectPyPI = %q, want %q", result, "great-tables")
	}
}

func TestDetectPyPIMissing(t *testing.T) {
	dir := t.TempDir()
	result := DetectPyPI(dir)
	if result != "" {
		t.Errorf("DetectPyPI = %q, want empty", result)
	}
}

func TestDetectPyPINoName(t *testing.T) {
	dir := t.TempDir()
	content := `[tool.setuptools]
packages = ["src"]
`
	os.WriteFile(filepath.Join(dir, "pyproject.toml"), []byte(content), 0644)

	result := DetectPyPI(dir)
	if result != "" {
		t.Errorf("DetectPyPI = %q, want empty", result)
	}
}

func TestDetectCRAN(t *testing.T) {
	dir := t.TempDir()
	content := `Package: dplyr
Title: A Grammar of Data Manipulation
Version: 1.1.4
Authors@R: person("Hadley", "Wickham")
`
	os.WriteFile(filepath.Join(dir, "DESCRIPTION"), []byte(content), 0644)

	result := DetectCRAN(dir)
	if result != "dplyr" {
		t.Errorf("DetectCRAN = %q, want %q", result, "dplyr")
	}
}

func TestDetectCRANMissing(t *testing.T) {
	dir := t.TempDir()
	result := DetectCRAN(dir)
	if result != "" {
		t.Errorf("DetectCRAN = %q, want empty", result)
	}
}

func TestDetectOpenVSX(t *testing.T) {
	dir := t.TempDir()
	content := `{
  "name": "positron",
  "publisher": "posit",
  "version": "1.0.0",
  "engines": {
    "vscode": "^1.80.0"
  }
}`
	os.WriteFile(filepath.Join(dir, "package.json"), []byte(content), 0644)

	result := DetectOpenVSX(dir)
	if result != "posit/positron" {
		t.Errorf("DetectOpenVSX = %q, want %q", result, "posit/positron")
	}
}

func TestDetectOpenVSXNoVSCode(t *testing.T) {
	dir := t.TempDir()
	content := `{
  "name": "my-app",
  "version": "1.0.0"
}`
	os.WriteFile(filepath.Join(dir, "package.json"), []byte(content), 0644)

	result := DetectOpenVSX(dir)
	if result != "" {
		t.Errorf("DetectOpenVSX = %q, want empty", result)
	}
}

func TestDetectOpenVSXMissing(t *testing.T) {
	dir := t.TempDir()
	result := DetectOpenVSX(dir)
	if result != "" {
		t.Errorf("DetectOpenVSX = %q, want empty", result)
	}
}

func TestDetectAll(t *testing.T) {
	dir := t.TempDir()

	// Create pyproject.toml
	os.WriteFile(filepath.Join(dir, "pyproject.toml"), []byte(`[project]
name = "my-pkg"
`), 0644)

	// Create DESCRIPTION
	os.WriteFile(filepath.Join(dir, "DESCRIPTION"), []byte(`Package: mypkg
Title: My Package
`), 0644)

	d := DetectAll(dir)

	if d.PyPI != "my-pkg" {
		t.Errorf("PyPI = %q, want %q", d.PyPI, "my-pkg")
	}
	if d.PyPISource != "pyproject.toml" {
		t.Errorf("PyPISource = %q, want %q", d.PyPISource, "pyproject.toml")
	}
	if d.CRAN != "mypkg" {
		t.Errorf("CRAN = %q, want %q", d.CRAN, "mypkg")
	}
	if d.CRANSource != "DESCRIPTION" {
		t.Errorf("CRANSource = %q, want %q", d.CRANSource, "DESCRIPTION")
	}
}
