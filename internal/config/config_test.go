package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadSingleProject(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "velocirepo.toml")
	content := `
[data]
dir = "metrics"

[projects.my-project]
name = "My Project"
github = "owner/repo"
pypi = "my-project"
`
	if err := os.WriteFile(cfgPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.Data.Dir != "metrics" {
		t.Errorf("Data.Dir = %q, want %q", cfg.Data.Dir, "metrics")
	}

	if cfg.DataDir() != filepath.Join(dir, "metrics") {
		t.Errorf("DataDir() = %q, want %q", cfg.DataDir(), filepath.Join(dir, "metrics"))
	}

	projects := cfg.ResolveProjects()
	if len(projects) != 1 {
		t.Fatalf("ResolveProjects() returned %d projects, want 1", len(projects))
	}

	p, ok := projects["my-project"]
	if !ok {
		t.Fatal("expected project with id 'my-project'")
	}
	if p.Name != "My Project" {
		t.Errorf("Name = %q, want %q", p.Name, "My Project")
	}
	if p.GitHub.First() != "owner/repo" {
		t.Errorf("GitHub = %q, want %q", p.GitHub.First(), "owner/repo")
	}
	if p.PyPI.First() != "my-project" {
		t.Errorf("PyPI = %q, want %q", p.PyPI.First(), "my-project")
	}
}

func TestLoadMultiProject(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "velocirepo.toml")
	content := `
[projects.alpha]
name = "Alpha"
github = "org/alpha"
pypi = "alpha"

[projects.beta]
name = "Beta"
github = "org/beta"
cran = "beta"
`
	if err := os.WriteFile(cfgPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	projects := cfg.ResolveProjects()
	if len(projects) != 2 {
		t.Fatalf("ResolveProjects() returned %d projects, want 2", len(projects))
	}

	if projects["alpha"].Name != "Alpha" {
		t.Errorf("alpha.Name = %q, want %q", projects["alpha"].Name, "Alpha")
	}
	if projects["beta"].CRAN.First() != "beta" {
		t.Errorf("beta.CRAN = %q, want %q", projects["beta"].CRAN.First(), "beta")
	}
}

func TestLoadMultiValueSources(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "velocirepo.toml")
	content := `
[projects.my-org]
name = "My Org"
github = ["org/repo-a", "org/repo-b"]
pypi = ["pkg-one", "pkg-two"]
cran = "single-pkg"
`
	if err := os.WriteFile(cfgPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	proj := cfg.ResolveProjects()["my-org"]
	if len(proj.GitHub) != 2 {
		t.Fatalf("GitHub has %d entries, want 2", len(proj.GitHub))
	}
	if proj.GitHub[0] != "org/repo-a" || proj.GitHub[1] != "org/repo-b" {
		t.Errorf("GitHub = %v, want [org/repo-a, org/repo-b]", proj.GitHub)
	}
	if len(proj.PyPI) != 2 {
		t.Fatalf("PyPI has %d entries, want 2", len(proj.PyPI))
	}
	if proj.CRAN.First() != "single-pkg" {
		t.Errorf("CRAN = %q, want %q", proj.CRAN.First(), "single-pkg")
	}
	if len(proj.CRAN) != 1 {
		t.Errorf("CRAN has %d entries, want 1", len(proj.CRAN))
	}
}

func TestLoadMissingFile(t *testing.T) {
	_, err := Load("/nonexistent/path/velocirepo.toml")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestLoadInvalidTOML(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "velocirepo.toml")
	if err := os.WriteFile(cfgPath, []byte("invalid [[[toml"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(cfgPath)
	if err == nil {
		t.Fatal("expected error for invalid TOML")
	}
}

func TestDiscovery(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "a", "b", "c")
	if err := os.MkdirAll(sub, 0755); err != nil {
		t.Fatal(err)
	}

	cfgPath := filepath.Join(dir, "velocirepo.toml")
	content := `[projects.found]
name = "Found"
github = "org/found"
`
	if err := os.WriteFile(cfgPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	if err := os.Chdir(sub); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load with discovery failed: %v", err)
	}

	projects := cfg.ResolveProjects()
	if projects["found"].Name != "Found" {
		t.Errorf("Name = %q, want %q", projects["found"].Name, "Found")
	}
}

func TestEnvVarOverride(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "velocirepo.toml")
	content := `[projects.envtest]
name = "EnvTest"
github = "org/envtest"
`
	if err := os.WriteFile(cfgPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("VELOCIREPO_CONFIG", cfgPath)

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load with env var failed: %v", err)
	}

	projects := cfg.ResolveProjects()
	if projects["envtest"].Name != "EnvTest" {
		t.Errorf("Name = %q, want %q", projects["envtest"].Name, "EnvTest")
	}
}

func TestDefaultDataDir(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "velocirepo.toml")
	content := `[projects.test]
name = "Test"
github = "org/test"
`
	if err := os.WriteFile(cfgPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	want := filepath.Join(dir, "velocirepo/data")
	if cfg.DataDir() != want {
		t.Errorf("DataDir() = %q, want %q", cfg.DataDir(), want)
	}
}

func TestNoProjects(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "velocirepo.toml")
	content := `[data]
dir = "data"
`
	if err := os.WriteFile(cfgPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	projects := cfg.ResolveProjects()
	if projects != nil {
		t.Errorf("expected nil projects, got %v", projects)
	}
}
