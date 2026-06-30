package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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
	assertContains(t, output, "alpha")
	assertContains(t, output, "beta")
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

	assertContains(t, buf.String(), "No projects")
}

func TestProjectAddWithFlags(t *testing.T) {
	cfgPath := setupTestConfig(t, `[data]
dir = "data"
`)

	_, _, err := execCmd(cfgPath, "add-project", "myproj", "--github", "org/myproj", "--pypi", "myproj")
	if err != nil {
		t.Fatal(err)
	}

	content := readFileString(t, cfgPath)
	assertContains(t, content, "[projects.myproj]")
	assertContains(t, content, `github = "org/myproj"`)
	assertContains(t, content, `pypi = "myproj"`)
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

	content := readFileString(t, cfgPath)
	assertNotContains(t, content, "alpha")
	assertContains(t, content, "[projects.beta]")
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
	cfgPath := setupTestConfig(t, `[data]
dir = "data"

[projects.alpha]
name = "Alpha"
github ="org/alpha"
`)
	dir := filepath.Dir(cfgPath)

	dataDir := filepath.Join(dir, "data", "events", "github", "alpha")
	_ = os.MkdirAll(dataDir, 0755)
	writeTempFile(t, dataDir, "2025-01-01.jsonl", `{}`)
	writeTempFile(t, filepath.Join(dir, "data"), ".schema-version", "5\n")

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

	content := readFileString(t, cfgPath)
	assertContains(t, content, `pypi = "alpha-pkg"`)
	assertContains(t, content, `github ="org/alpha"`)
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

	content := readFileString(t, cfgPath)
	assertNotContains(t, content, "pypi")
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
	cfgPath := setupTestConfig(t, `[data]
dir = "data"

[projects.alpha]
name = "Alpha"
github ="org/alpha"
`)
	dir := filepath.Dir(cfgPath)

	dataDir := filepath.Join(dir, "data", "events", "github", "alpha")
	_ = os.MkdirAll(dataDir, 0755)
	writeTempFile(t, dataDir, "2025-01-01.jsonl", `{"source":"github","event_type":"star","project_id":"alpha","github_repo":"org/alpha","datetime":"2025-01-01T10:00:00Z","user":"alice"}`+"\n")
	writeTempFile(t, filepath.Join(dir, "data"), ".schema-version", "5\n")

	_, buf, err := execCmd(cfgPath, "show-project", "alpha")
	if err != nil {
		t.Fatal(err)
	}

	output := buf.String()
	assertContains(t, output, "Alpha")
	assertContains(t, output, "2025-01-01")
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
	cfgPath := setupTestConfig(t, `[data]
dir = "data"

[projects.alpha]
name = "Alpha"
github ="org/alpha"
`)

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
	cfgPath := setupTestConfig(t, `[data]
dir = "data"

[projects.old-name]
name = "Old"
github ="org/old"
`)
	dir := filepath.Dir(cfgPath)

	dataDir := filepath.Join(dir, "data", "events", "github", "old-name")
	_ = os.MkdirAll(dataDir, 0755)
	writeTempFile(t, dataDir, "2025-01-01.jsonl", `{"source":"github","type":"star","project_id":"old-name","target":"org/old","datetime":"2025-01-01T00:00:00Z"}`+"\n")
	writeTempFile(t, filepath.Join(dir, "data"), ".schema-version", "5\n")

	_, _, err := execCmd(cfgPath, "rename-project", "old-name", "new-name")
	if err != nil {
		t.Fatal(err)
	}

	content := readFileString(t, cfgPath)
	assertNotContains(t, content, "old-name")
	assertContains(t, content, "[projects.new-name]")

	newDataDir := filepath.Join(dir, "data", "events", "github", "new-name")
	if _, err := os.Stat(newDataDir); os.IsNotExist(err) {
		t.Error("data directory not moved")
	}
	renamedData := readFileString(t, filepath.Join(newDataDir, "2025-01-01.jsonl"))
	assertContains(t, renamedData, `"project_id":"new-name"`)
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
	cfgPath := setupTestConfig(t, `[data]
dir = "data"
`)
	dir := filepath.Dir(cfgPath)

	importFile := writeTempFile(t, dir, "projects.json", `[
		{"id": "proj-a", "github": "org/proj-a"},
		{"id": "proj-b", "github": "org/proj-b", "pypi": "proj-b"}
	]`)

	_, _, err := execCmd(cfgPath, "import-projects", "--from-file", importFile, "--yes")
	if err != nil {
		t.Fatal(err)
	}

	content := readFileString(t, cfgPath)
	assertContains(t, content, "[projects.proj-a]")
	assertContains(t, content, "[projects.proj-b]")
	assertContains(t, content, `pypi = "proj-b"`)
}

func TestProjectImportFromCSV(t *testing.T) {
	cfgPath := setupTestConfig(t, `[data]
dir = "data"
`)
	dir := filepath.Dir(cfgPath)

	importFile := writeTempFile(t, dir, "projects.csv", "id,name,github,pypi\nmy-proj,My Project,org/my-proj,my-proj\n")

	_, _, err := execCmd(cfgPath, "import-projects", "--from-file", importFile, "--yes")
	if err != nil {
		t.Fatal(err)
	}

	content := readFileString(t, cfgPath)
	assertContains(t, content, "[projects.my-proj]")
}

func TestProjectImportTrafficAliasFallback(t *testing.T) {
	cfgPath := setupTestConfig(t, `[data]
dir = "data"
`)
	dir := filepath.Dir(cfgPath)

	jsonFile := writeTempFile(t, dir, "projects.json", `[
		{"id": "json-proj", "github_traffic": "", "github-traffic": "org/json-proj"}
	]`)
	if _, _, err := execCmd(cfgPath, "import-projects", "--from-file", jsonFile, "--yes"); err != nil {
		t.Fatal(err)
	}

	csvFile := writeTempFile(t, dir, "projects.csv", "id,github-traffic,github_traffic\ncsv-proj, ,org/csv-proj\n")
	if _, _, err := execCmd(cfgPath, "import-projects", "--from-file", csvFile, "--yes"); err != nil {
		t.Fatal(err)
	}

	content := readFileString(t, cfgPath)
	assertContains(t, content, `github-traffic = "org/json-proj"`)
	assertContains(t, content, `github-traffic = "org/csv-proj"`)
}

func TestProjectImportDryRun(t *testing.T) {
	cfgPath := setupTestConfig(t, `[data]
dir = "data"
`)
	dir := filepath.Dir(cfgPath)

	importFile := writeTempFile(t, dir, "projects.json", `[{"id": "proj-a", "github": "org/proj-a"}]`)

	_, _, err := execCmd(cfgPath, "import-projects", "--from-file", importFile, "--dry-run")
	if err != nil {
		t.Fatal(err)
	}

	content := readFileString(t, cfgPath)
	assertNotContains(t, content, "proj-a")
}

func TestProjectImportSkipExisting(t *testing.T) {
	cfgPath := setupTestConfig(t, `[data]
dir = "data"

[projects.existing]
name = "Existing"
github ="org/existing"
`)
	dir := filepath.Dir(cfgPath)

	importFile := writeTempFile(t, dir, "projects.json", `[
		{"id": "existing", "github": "org/existing"},
		{"id": "new-proj", "github": "org/new-proj"}
	]`)

	_, _, err := execCmd(cfgPath, "import-projects", "--from-file", importFile, "--skip-existing", "--yes")
	if err != nil {
		t.Fatal(err)
	}

	content := readFileString(t, cfgPath)
	assertContains(t, content, "[projects.new-proj]")
}

func TestProjectInitCreatesConfig(t *testing.T) {
	dir := t.TempDir()

	rootCmd := newRootCmd()
	rootCmd.SetArgs([]string{"init", "--dir", dir})
	if err := rootCmd.Execute(); err != nil {
		t.Fatal(err)
	}

	cfgPath := filepath.Join(dir, "velocirepo.toml")
	content := readFileString(t, cfgPath)
	assertContains(t, content, `dir = "velocirepo/data"`)

	dataDir := filepath.Join(dir, "velocirepo/data")
	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		t.Error("data directory not created")
	}
}

func TestProjectInitAlreadyExists(t *testing.T) {
	dir := t.TempDir()
	writeTempFile(t, dir, "velocirepo.toml", "[data]\n")

	rootCmd := newRootCmd()
	rootCmd.SetArgs([]string{"init", "--dir", dir})
	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error when config already exists")
	}
}
