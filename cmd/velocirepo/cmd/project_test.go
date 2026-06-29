package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func setupTestConfig(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "velocirepo.toml")
	if err := os.WriteFile(cfgPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return cfgPath
}

func execCmd(cfgPath string, args ...string) (*cobra.Command, *bytes.Buffer, error) {
	rootCmd := newRootCmd()
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	fullArgs := append([]string{"--config", cfgPath}, args...)
	rootCmd.SetArgs(fullArgs)
	err := rootCmd.Execute()
	return rootCmd, buf, err
}

func TestProjectListTable(t *testing.T) {
	cfgPath := setupTestConfig(t, `[projects.alpha]
name = "Alpha"
github ="org/alpha"

[projects.beta]
name = "Beta"
pypi = "beta-pkg"
`)

	_, buf, err := execCmd(cfgPath, "list-projects")
	if err != nil {
		t.Fatal(err)
	}

	output := buf.String()
	if !strings.Contains(output, "alpha") {
		t.Errorf("output missing 'alpha': %s", output)
	}
	if !strings.Contains(output, "beta") {
		t.Errorf("output missing 'beta': %s", output)
	}
}

func TestProjectListJSON(t *testing.T) {
	cfgPath := setupTestConfig(t, `[projects.myproj]
name = "My Project"
github ="me/myproj"
`)

	_, buf, err := execCmd(cfgPath, "list-projects", "--json")
	if err != nil {
		t.Fatal(err)
	}

	var list []map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &list); err != nil {
		t.Fatalf("invalid JSON output: %v\n%s", err, buf.String())
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 project, got %d", len(list))
	}
	if list[0]["id"] != "myproj" {
		t.Errorf("id = %v, want 'myproj'", list[0]["id"])
	}
}

func TestProjectListQuiet(t *testing.T) {
	cfgPath := setupTestConfig(t, `[projects.alpha]
name = "Alpha"
github ="org/alpha"

[projects.beta]
name = "Beta"
github ="org/beta"
`)

	_, buf, err := execCmd(cfgPath, "list-projects", "--ids-only")
	if err != nil {
		t.Fatal(err)
	}

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d: %v", len(lines), lines)
	}
}

func TestProjectListEmpty(t *testing.T) {
	cfgPath := setupTestConfig(t, `[data]
dir = "data"
`)

	_, buf, err := execCmd(cfgPath, "list-projects")
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(buf.String(), "No projects") {
		t.Errorf("expected 'No projects' message, got: %s", buf.String())
	}
}

func TestProjectAddWithFlags(t *testing.T) {
	cfgPath := setupTestConfig(t, `[data]
dir = "data"
`)

	_, _, err := execCmd(cfgPath, "add-project", "myproj", "--github", "org/myproj", "--pypi", "myproj")
	if err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(cfgPath)
	content := string(data)
	if !strings.Contains(content, "[projects.myproj]") {
		t.Error("project section not added")
	}
	if !strings.Contains(content, `github = "org/myproj"`) {
		t.Error("github field missing")
	}
	if !strings.Contains(content, `pypi = "myproj"`) {
		t.Error("pypi field missing")
	}
}

func TestProjectAddDuplicate(t *testing.T) {
	cfgPath := setupTestConfig(t, `[projects.existing]
name = "Existing"
github ="org/existing"
`)

	_, _, err := execCmd(cfgPath, "add-project", "existing", "--github", "org/other")
	if err == nil {
		t.Fatal("expected error for duplicate project")
	}
}

func TestProjectAddInvalidID(t *testing.T) {
	cfgPath := setupTestConfig(t, `[data]
dir = "data"
`)

	_, _, err := execCmd(cfgPath, "add-project", "INVALID_ID", "--github", "org/repo")
	if err == nil {
		t.Fatal("expected error for invalid ID")
	}
}

func TestProjectAddInvalidGitHub(t *testing.T) {
	cfgPath := setupTestConfig(t, `[data]
dir = "data"
`)

	_, _, err := execCmd(cfgPath, "add-project", "myproj", "--github", "not-a-valid-format")
	if err == nil {
		t.Fatal("expected error for invalid GitHub format")
	}
}

func TestProjectRemoveForce(t *testing.T) {
	cfgPath := setupTestConfig(t, `[projects.alpha]
name = "Alpha"
github ="org/alpha"

[projects.beta]
name = "Beta"
github ="org/beta"
`)

	_, _, err := execCmd(cfgPath, "remove-project", "alpha", "--force")
	if err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(cfgPath)
	content := string(data)
	if strings.Contains(content, "alpha") {
		t.Error("alpha still present after removal")
	}
	if !strings.Contains(content, "[projects.beta]") {
		t.Error("beta should still exist")
	}
}

func TestProjectRemoveNotFound(t *testing.T) {
	cfgPath := setupTestConfig(t, `[projects.alpha]
name = "Alpha"
github ="org/alpha"
`)

	_, _, err := execCmd(cfgPath, "remove-project", "nonexistent", "--force")
	if err == nil {
		t.Fatal("expected error for nonexistent project")
	}
}

func TestProjectRemoveWithDeleteData(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "velocirepo.toml")
	os.WriteFile(cfgPath, []byte(`[data]
dir = "data"

[projects.alpha]
name = "Alpha"
github ="org/alpha"
`), 0644)

	dataDir := filepath.Join(dir, "data", "events", "github", "alpha")
	os.MkdirAll(dataDir, 0755)
	os.WriteFile(filepath.Join(dataDir, "2025-01-01.jsonl"), []byte(`{}`), 0644)
	os.WriteFile(filepath.Join(dir, "data", ".schema-version"), []byte("5\n"), 0644)

	_, _, err := execCmd(cfgPath, "remove-project", "alpha", "--force", "--delete-data")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(dataDir); !os.IsNotExist(err) {
		t.Error("data directory should have been deleted")
	}
}

func TestProjectUpdateFlags(t *testing.T) {
	cfgPath := setupTestConfig(t, `[projects.alpha]
name = "Alpha"
github ="org/alpha"
`)

	_, _, err := execCmd(cfgPath, "update-project", "alpha", "--pypi", "alpha-pkg")
	if err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(cfgPath)
	content := string(data)
	if !strings.Contains(content, `pypi = "alpha-pkg"`) {
		t.Errorf("pypi not added:\n%s", content)
	}
	if !strings.Contains(content, `github ="org/alpha"`) {
		t.Error("github field lost")
	}
}

func TestProjectUpdateUnset(t *testing.T) {
	cfgPath := setupTestConfig(t, `[projects.alpha]
name = "Alpha"
github ="org/alpha"
pypi = "alpha"
`)

	_, _, err := execCmd(cfgPath, "update-project", "alpha", "--unset", "pypi")
	if err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(cfgPath)
	content := string(data)
	if strings.Contains(content, "pypi") {
		t.Errorf("pypi still present:\n%s", content)
	}
}

func TestProjectUpdateNotFound(t *testing.T) {
	cfgPath := setupTestConfig(t, `[projects.alpha]
name = "Alpha"
github ="org/alpha"
`)

	_, _, err := execCmd(cfgPath, "update-project", "nonexistent", "--name", "New")
	if err == nil {
		t.Fatal("expected error for nonexistent project")
	}
}

func TestProjectShow(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "velocirepo.toml")
	os.WriteFile(cfgPath, []byte(`[data]
dir = "data"

[projects.alpha]
name = "Alpha"
github ="org/alpha"
`), 0644)

	dataDir := filepath.Join(dir, "data", "events", "github", "alpha")
	os.MkdirAll(dataDir, 0755)
	os.WriteFile(filepath.Join(dataDir, "2025-01-01.jsonl"),
		[]byte(`{"source":"github","event_type":"star","project_id":"alpha","github_repo":"org/alpha","datetime":"2025-01-01T10:00:00Z","user":"alice"}`+"\n"), 0644)
	os.WriteFile(filepath.Join(dir, "data", ".schema-version"), []byte("5\n"), 0644)

	_, buf, err := execCmd(cfgPath, "show-project", "alpha")
	if err != nil {
		t.Fatal(err)
	}

	output := buf.String()
	if !strings.Contains(output, "Alpha") {
		t.Errorf("output missing project name: %s", output)
	}
	if !strings.Contains(output, "2025-01-01") {
		t.Errorf("output missing last date: %s", output)
	}
}

func TestProjectShowNotFound(t *testing.T) {
	cfgPath := setupTestConfig(t, `[projects.alpha]
name = "Alpha"
github ="org/alpha"
`)

	_, _, err := execCmd(cfgPath, "show-project", "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent project")
	}
}

func TestProjectShowJSON(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "velocirepo.toml")
	os.WriteFile(cfgPath, []byte(`[data]
dir = "data"

[projects.alpha]
name = "Alpha"
github ="org/alpha"
`), 0644)

	_, buf, err := execCmd(cfgPath, "show-project", "alpha", "--json")
	if err != nil {
		t.Fatal(err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, buf.String())
	}
	if result["id"] != "alpha" {
		t.Errorf("id = %v, want 'alpha'", result["id"])
	}
}

func TestProjectRename(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "velocirepo.toml")
	os.WriteFile(cfgPath, []byte(`[data]
dir = "data"

[projects.old-name]
name = "Old"
github ="org/old"
`), 0644)

	dataDir := filepath.Join(dir, "data", "events", "github", "old-name")
	os.MkdirAll(dataDir, 0755)
	os.WriteFile(filepath.Join(dataDir, "2025-01-01.jsonl"), []byte(`{"source":"github","type":"star","project_id":"old-name","target":"org/old","datetime":"2025-01-01T00:00:00Z"}`+"\n"), 0644)
	os.WriteFile(filepath.Join(dir, "data", ".schema-version"), []byte("5\n"), 0644)

	_, _, err := execCmd(cfgPath, "rename-project", "old-name", "new-name")
	if err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(cfgPath)
	content := string(data)
	if strings.Contains(content, "old-name") {
		t.Error("old name still in config")
	}
	if !strings.Contains(content, "[projects.new-name]") {
		t.Error("new name not in config")
	}

	newDataDir := filepath.Join(dir, "data", "events", "github", "new-name")
	if _, err := os.Stat(newDataDir); os.IsNotExist(err) {
		t.Error("data directory not moved")
	}
	renamedData, _ := os.ReadFile(filepath.Join(newDataDir, "2025-01-01.jsonl"))
	if !strings.Contains(string(renamedData), `"project_id":"new-name"`) {
		t.Errorf("project_id not rewritten in renamed data: %s", renamedData)
	}
	if _, err := os.Stat(dataDir); !os.IsNotExist(err) {
		t.Error("old data directory still exists")
	}
}

func TestProjectRenameNoMoveData(t *testing.T) {
	cfgPath := setupTestConfig(t, `[projects.old-name]
name = "Old"
github ="org/old"
`)

	_, _, err := execCmd(cfgPath, "rename-project", "old-name", "new-name", "--no-move-data")
	if err != nil {
		t.Fatal(err)
	}
}

func TestProjectRenameInvalidNewID(t *testing.T) {
	cfgPath := setupTestConfig(t, `[projects.alpha]
name = "Alpha"
github ="org/alpha"
`)

	_, _, err := execCmd(cfgPath, "rename-project", "alpha", "INVALID")
	if err == nil {
		t.Fatal("expected error for invalid new ID")
	}
}

func TestProjectRenameNewIDExists(t *testing.T) {
	cfgPath := setupTestConfig(t, `[projects.alpha]
name = "Alpha"
github ="org/alpha"

[projects.beta]
name = "Beta"
github ="org/beta"
`)

	_, _, err := execCmd(cfgPath, "rename-project", "alpha", "beta")
	if err == nil {
		t.Fatal("expected error when new ID already exists")
	}
}

func TestProjectImportFromJSON(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "velocirepo.toml")
	os.WriteFile(cfgPath, []byte(`[data]
dir = "data"
`), 0644)

	importFile := filepath.Join(dir, "projects.json")
	os.WriteFile(importFile, []byte(`[
		{"id": "proj-a", "github": "org/proj-a"},
		{"id": "proj-b", "github": "org/proj-b", "pypi": "proj-b"}
	]`), 0644)

	_, _, err := execCmd(cfgPath, "import-projects", "--from-file", importFile, "--yes")
	if err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(cfgPath)
	content := string(data)
	if !strings.Contains(content, "[projects.proj-a]") {
		t.Error("proj-a not added")
	}
	if !strings.Contains(content, "[projects.proj-b]") {
		t.Error("proj-b not added")
	}
	if !strings.Contains(content, `pypi = "proj-b"`) {
		t.Error("pypi field missing for proj-b")
	}
}

func TestProjectImportFromCSV(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "velocirepo.toml")
	os.WriteFile(cfgPath, []byte(`[data]
dir = "data"
`), 0644)

	importFile := filepath.Join(dir, "projects.csv")
	os.WriteFile(importFile, []byte("id,name,github,pypi\nmy-proj,My Project,org/my-proj,my-proj\n"), 0644)

	_, _, err := execCmd(cfgPath, "import-projects", "--from-file", importFile, "--yes")
	if err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(cfgPath)
	content := string(data)
	if !strings.Contains(content, "[projects.my-proj]") {
		t.Error("project not added from CSV")
	}
}

func TestProjectImportDryRun(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "velocirepo.toml")
	os.WriteFile(cfgPath, []byte(`[data]
dir = "data"
`), 0644)

	importFile := filepath.Join(dir, "projects.json")
	os.WriteFile(importFile, []byte(`[{"id": "proj-a", "github": "org/proj-a"}]`), 0644)

	_, _, err := execCmd(cfgPath, "import-projects", "--from-file", importFile, "--dry-run")
	if err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(cfgPath)
	content := string(data)
	if strings.Contains(content, "proj-a") {
		t.Error("dry-run should not modify config")
	}
}

func TestProjectImportSkipExisting(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "velocirepo.toml")
	os.WriteFile(cfgPath, []byte(`[data]
dir = "data"

[projects.existing]
name = "Existing"
github ="org/existing"
`), 0644)

	importFile := filepath.Join(dir, "projects.json")
	os.WriteFile(importFile, []byte(`[
		{"id": "existing", "github": "org/existing"},
		{"id": "new-proj", "github": "org/new-proj"}
	]`), 0644)

	_, _, err := execCmd(cfgPath, "import-projects", "--from-file", importFile, "--skip-existing", "--yes")
	if err != nil {
		t.Fatal(err)
	}

	data, _ := os.ReadFile(cfgPath)
	content := string(data)
	if !strings.Contains(content, "[projects.new-proj]") {
		t.Error("new-proj should have been added")
	}
}

func TestProjectInitCreatesConfig(t *testing.T) {
	dir := t.TempDir()

	rootCmd := newRootCmd()
	rootCmd.SetArgs([]string{"init", "--dir", dir})
	if err := rootCmd.Execute(); err != nil {
		t.Fatal(err)
	}

	cfgPath := filepath.Join(dir, "velocirepo.toml")
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatal("config file not created")
	}
	content := string(data)
	if !strings.Contains(content, `dir = "velocirepo/data"`) {
		t.Errorf("missing data dir in config:\n%s", content)
	}

	dataDir := filepath.Join(dir, "velocirepo/data")
	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		t.Error("data directory not created")
	}
}

func TestProjectInitAlreadyExists(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "velocirepo.toml"), []byte("[data]\n"), 0644)

	rootCmd := newRootCmd()
	rootCmd.SetArgs([]string{"init", "--dir", dir})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error when config already exists")
	}
}
