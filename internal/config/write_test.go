package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFindSection(t *testing.T) {
	lines := []string{
		"[data]",
		`dir = "data"`,
		"",
		"[projects.alpha]",
		`name = "Alpha"`,
		`github = "org/alpha"`,
		"",
		"[projects.beta]",
		`name = "Beta"`,
	}

	start, end, found := FindSection(lines, "projects.alpha")
	if !found {
		t.Fatal("expected to find section")
	}
	if start != 3 {
		t.Errorf("start = %d, want 3", start)
	}
	if end != 7 {
		t.Errorf("end = %d, want 7", end)
	}

	start, end, found = FindSection(lines, "projects.beta")
	if !found {
		t.Fatal("expected to find section")
	}
	if start != 7 {
		t.Errorf("start = %d, want 7", start)
	}
	if end != 9 {
		t.Errorf("end = %d, want 9 (len)", end)
	}

	_, _, found = FindSection(lines, "projects.missing")
	if found {
		t.Error("expected not found for missing section")
	}
}

func TestAppendProject(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "velocirepo.toml")
	initial := "[data]\ndir = \"data\"\n\n[projects.alpha]\nname = \"Alpha\"\n"
	os.WriteFile(path, []byte(initial), 0644)

	err := AppendProject(path, "beta", Project{
		Name:   "Beta",
		GitHub: StringList{"org/beta"},
		PyPI:   StringList{"beta-pkg"},
	})
	if err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(path)
	content := string(data)

	if !strings.Contains(content, "[projects.beta]") {
		t.Error("missing [projects.beta] section")
	}
	if !strings.Contains(content, `github = "org/beta"`) {
		t.Error("missing github field")
	}
	if !strings.Contains(content, `pypi = "beta-pkg"`) {
		t.Error("missing pypi field")
	}
	// Original content preserved
	if !strings.Contains(content, "[projects.alpha]") {
		t.Error("original section lost")
	}
}

func TestAppendProjectPreservesComments(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "velocirepo.toml")
	initial := "# Main config\n[data]\ndir = \"data\"\n\n# Alpha project\n[projects.alpha]\nname = \"Alpha\"\n"
	os.WriteFile(path, []byte(initial), 0644)

	err := AppendProject(path, "beta", Project{Name: "Beta", GitHub: StringList{"org/beta"}})
	if err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(path)
	content := string(data)

	if !strings.Contains(content, "# Main config") {
		t.Error("lost top comment")
	}
	if !strings.Contains(content, "# Alpha project") {
		t.Error("lost section comment")
	}
}

func TestRemoveProjectMiddle(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "velocirepo.toml")
	initial := `[data]
dir = "data"

[projects.alpha]
name = "Alpha"

[projects.beta]
name = "Beta"

[projects.gamma]
name = "Gamma"
`
	os.WriteFile(path, []byte(initial), 0644)

	err := RemoveProject(path, "beta")
	if err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(path)
	content := string(data)

	if strings.Contains(content, "beta") {
		t.Error("beta section still present")
	}
	if !strings.Contains(content, "[projects.alpha]") {
		t.Error("alpha section lost")
	}
	if !strings.Contains(content, "[projects.gamma]") {
		t.Error("gamma section lost")
	}
}

func TestRemoveProjectEnd(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "velocirepo.toml")
	initial := `[data]
dir = "data"

[projects.alpha]
name = "Alpha"

[projects.beta]
name = "Beta"
`
	os.WriteFile(path, []byte(initial), 0644)

	err := RemoveProject(path, "beta")
	if err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(path)
	content := string(data)

	if strings.Contains(content, "beta") {
		t.Error("beta section still present")
	}
	if !strings.Contains(content, "[projects.alpha]") {
		t.Error("alpha section lost")
	}
}

func TestRemoveProjectNotFound(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "velocirepo.toml")
	os.WriteFile(path, []byte("[data]\ndir = \"data\"\n"), 0644)

	err := RemoveProject(path, "nope")
	if err == nil {
		t.Fatal("expected error for missing project")
	}
}

func TestUpdateProjectModifyField(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "velocirepo.toml")
	initial := `[projects.alpha]
name = "Alpha"
github = "org/alpha"
pypi = "alpha"
`
	os.WriteFile(path, []byte(initial), 0644)

	err := UpdateProject(path, "alpha", map[string]string{"name": "Alpha v2"}, nil)
	if err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(path)
	content := string(data)

	if !strings.Contains(content, `name = "Alpha v2"`) {
		t.Errorf("name not updated, got:\n%s", content)
	}
	if !strings.Contains(content, `github = "org/alpha"`) {
		t.Error("github field lost")
	}
}

func TestUpdateProjectAddField(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "velocirepo.toml")
	initial := `[projects.alpha]
name = "Alpha"
github = "org/alpha"
`
	os.WriteFile(path, []byte(initial), 0644)

	err := UpdateProject(path, "alpha", map[string]string{"pypi": "alpha-pkg"}, nil)
	if err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(path)
	content := string(data)

	if !strings.Contains(content, `pypi = "alpha-pkg"`) {
		t.Errorf("pypi not added, got:\n%s", content)
	}
}

func TestUpdateProjectUnset(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "velocirepo.toml")
	initial := `[projects.alpha]
name = "Alpha"
github = "org/alpha"
pypi = "alpha"
`
	os.WriteFile(path, []byte(initial), 0644)

	err := UpdateProject(path, "alpha", nil, []string{"pypi"})
	if err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(path)
	content := string(data)

	if strings.Contains(content, "pypi") {
		t.Errorf("pypi still present, got:\n%s", content)
	}
	if !strings.Contains(content, `github = "org/alpha"`) {
		t.Error("github field lost")
	}
}

func TestUpdateProjectNotFound(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "velocirepo.toml")
	os.WriteFile(path, []byte("[data]\ndir = \"data\"\n"), 0644)

	err := UpdateProject(path, "nope", map[string]string{"name": "x"}, nil)
	if err == nil {
		t.Fatal("expected error for missing project")
	}
}

func TestRenameSection(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "velocirepo.toml")
	initial := `[projects.old-name]
name = "Old"
github = "org/old"
`
	os.WriteFile(path, []byte(initial), 0644)

	err := RenameSection(path, "old-name", "new-name")
	if err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(path)
	content := string(data)

	if strings.Contains(content, "old-name") {
		t.Error("old header still present")
	}
	if !strings.Contains(content, "[projects.new-name]") {
		t.Error("new header not found")
	}
	if !strings.Contains(content, `github = "org/old"`) {
		t.Error("body content lost")
	}
}

func TestRenameSectionNotFound(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "velocirepo.toml")
	os.WriteFile(path, []byte("[data]\ndir = \"data\"\n"), 0644)

	err := RenameSection(path, "nope", "new")
	if err == nil {
		t.Fatal("expected error for missing section")
	}
}
