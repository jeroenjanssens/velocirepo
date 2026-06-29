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
github ="owner/repo"
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

	if len(cfg.Projects) != 1 {
		t.Fatalf("Projects has %d entries, want 1", len(cfg.Projects))
	}

	p, ok := cfg.Projects["my-project"]
	if !ok {
		t.Fatal("expected project with id 'my-project'")
	}
	if p.Name != "My Project" {
		t.Errorf("Name = %q, want %q", p.Name, "My Project")
	}
	if p.GitHubEvents.First() != "owner/repo" {
		t.Errorf("GitHubEvents = %q, want %q", p.GitHubEvents.First(), "owner/repo")
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
github ="org/alpha"
pypi = "alpha"

[projects.beta]
name = "Beta"
github ="org/beta"
cran = "beta"
`
	if err := os.WriteFile(cfgPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if len(cfg.Projects) != 2 {
		t.Fatalf("Projects has %d entries, want 2", len(cfg.Projects))
	}

	if cfg.Projects["alpha"].Name != "Alpha" {
		t.Errorf("alpha.Name = %q, want %q", cfg.Projects["alpha"].Name, "Alpha")
	}
	if cfg.Projects["beta"].CRAN.First() != "beta" {
		t.Errorf("beta.CRAN = %q, want %q", cfg.Projects["beta"].CRAN.First(), "beta")
	}
}

func TestLoadMultiValueSources(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "velocirepo.toml")
	content := `
[projects.my-org]
name = "My Org"
github =["org/repo-a", "org/repo-b"]
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

	proj := cfg.Projects["my-org"]
	if len(proj.GitHubEvents) != 2 {
		t.Fatalf("GitHubEvents has %d entries, want 2", len(proj.GitHubEvents))
	}
	if proj.GitHubEvents[0] != "org/repo-a" || proj.GitHubEvents[1] != "org/repo-b" {
		t.Errorf("GitHubEvents = %v, want [org/repo-a, org/repo-b]", proj.GitHubEvents)
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

func TestProjectSourcesIncludeAllRegisteredSources(t *testing.T) {
	p := Project{
		GitHubEvents:  StringList{"org/repo"},
		GitHubTraffic: StringList{"org/repo"},
		PyPI:          StringList{"pkg"},
		CRAN:          StringList{"cranpkg"},
		Homebrew:      StringList{"tap/formula"},
		Plausible:     StringList{"example.com"},
		OpenVSX:       StringList{"pub/ext"},
		YouTube:       StringList{"@channel"},
		LinkedIn:      StringList{"urn:li:organization:123"},
	}

	got := p.SourceNames()
	want := []string{"github", "github-traffic", "pypi", "cran", "homebrew", "plausible", "openvsx", "youtube", "linkedin"}
	if len(got) != len(want) {
		t.Fatalf("SourceNames length = %d, want %d: %v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("SourceNames[%d] = %q, want %q (all: %v)", i, got[i], want[i], got)
		}
	}
}

func TestSourceDirNamesIncludeMetricsEventsAndContent(t *testing.T) {
	got := map[string]bool{}
	for _, path := range SourceDirNames() {
		got[path] = true
	}

	for _, want := range []string{
		"events/github",
		"metrics/github-traffic",
		"metrics/youtube",
		"metrics/linkedin",
		"content/youtube",
		"content/linkedin",
	} {
		if !got[want] {
			t.Fatalf("SourceDirNames missing %q: %v", want, SourceDirNames())
		}
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

	if cfg.Projects["found"].Name != "Found" {
		t.Errorf("Name = %q, want %q", cfg.Projects["found"].Name, "Found")
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

	if cfg.Projects["envtest"].Name != "EnvTest" {
		t.Errorf("Name = %q, want %q", cfg.Projects["envtest"].Name, "EnvTest")
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

	if len(cfg.Projects) != 0 {
		t.Errorf("expected empty projects, got %v", cfg.Projects)
	}
}
