package store

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSchemaVersionMissing(t *testing.T) {
	dir := t.TempDir()
	v, err := SchemaVersion(dir)
	if err != nil {
		t.Fatal(err)
	}
	if v != 0 {
		t.Errorf("expected version 0 for missing file, got %d", v)
	}
}

func TestSchemaVersionReadWrite(t *testing.T) {
	dir := t.TempDir()
	if err := writeSchemaVersion(dir, 3); err != nil {
		t.Fatal(err)
	}
	v, err := SchemaVersion(dir)
	if err != nil {
		t.Fatal(err)
	}
	if v != 3 {
		t.Errorf("expected version 3, got %d", v)
	}
}

func TestCheckSchemaVersionNoDataDir(t *testing.T) {
	err := CheckSchemaVersion("/nonexistent/path")
	if err != nil {
		t.Errorf("expected nil for nonexistent dir, got: %v", err)
	}
}

func TestCheckSchemaVersionStale(t *testing.T) {
	dir := t.TempDir()
	writeSchemaVersion(dir, 0)
	err := CheckSchemaVersion(dir)
	if err == nil {
		t.Fatal("expected error for stale schema")
	}
}

func TestCheckSchemaVersionCurrent(t *testing.T) {
	dir := t.TempDir()
	writeSchemaVersion(dir, LatestSchemaVersion)
	err := CheckSchemaVersion(dir)
	if err != nil {
		t.Errorf("expected nil for current schema, got: %v", err)
	}
}

func TestCheckSchemaVersionFuture(t *testing.T) {
	dir := t.TempDir()
	writeSchemaVersion(dir, LatestSchemaVersion+1)
	err := CheckSchemaVersion(dir)
	if err == nil {
		t.Fatal("expected error for future schema")
	}
}

func TestMigrate0to1(t *testing.T) {
	dir := t.TempDir()

	// Set up pypi data with old metric names
	pypiDir := filepath.Join(dir, "pypi", "myproj")
	os.MkdirAll(pypiDir, 0755)
	os.WriteFile(filepath.Join(pypiDir, "2025-06-01.jsonl"),
		[]byte(`{"source":"pypi","metric":"downloads","project_id":"myproj","date":"2025-06-01","value":100}`+"\n"), 0644)

	// Set up plausible data
	plausibleDir := filepath.Join(dir, "plausible", "myproj")
	os.MkdirAll(plausibleDir, 0755)
	os.WriteFile(filepath.Join(plausibleDir, "2025-06-01.jsonl"),
		[]byte(`{"source":"plausible","metric":"pageviews","project_id":"myproj","date":"2025-06-01","value":500}`+"\n"+
			`{"source":"plausible","metric":"visitors","project_id":"myproj","date":"2025-06-01","value":200}`+"\n"), 0644)

	// Set up openvsx data
	openvsxDir := filepath.Join(dir, "openvsx", "myproj")
	os.MkdirAll(openvsxDir, 0755)
	os.WriteFile(filepath.Join(openvsxDir, "2025-06-01.jsonl"),
		[]byte(`{"source":"openvsx","metric":"total_downloads","project_id":"myproj","date":"2025-06-01","value":5000}`+"\n"+
			`{"source":"openvsx","metric":"reviews","project_id":"myproj","date":"2025-06-01","value":10}`+"\n"+
			`{"source":"openvsx","metric":"rating","project_id":"myproj","date":"2025-06-01","value":450}`+"\n"), 0644)

	applied, err := Migrate(dir)
	if err != nil {
		t.Fatal(err)
	}
	if applied != 4 {
		t.Errorf("expected 4 migrations applied, got %d", applied)
	}

	v, _ := SchemaVersion(dir)
	if v != 4 {
		t.Errorf("expected schema version 4, got %d", v)
	}

	// Verify pypi (v4 moves it into metrics/)
	data, _ := os.ReadFile(filepath.Join(dir, "metrics", "pypi", "myproj", "2025-06-01.jsonl"))
	if got := string(data); !contains(got, `"daily_downloads"`) {
		t.Errorf("pypi not migrated: %s", got)
	}

	// Verify plausible (v4 moves it into metrics/)
	data, _ = os.ReadFile(filepath.Join(dir, "metrics", "plausible", "myproj", "2025-06-01.jsonl"))
	content := string(data)
	if !contains(content, `"daily_site_pageviews"`) || !contains(content, `"daily_site_visitors"`) {
		t.Errorf("plausible not migrated: %s", content)
	}

	// Verify openvsx (v4 moves it into metrics/)
	data, _ = os.ReadFile(filepath.Join(dir, "metrics", "openvsx", "myproj", "2025-06-01.jsonl"))
	content = string(data)
	if !contains(content, `"total_downloads"`) || !contains(content, `"total_reviews"`) || !contains(content, `"total_ratings"`) {
		t.Errorf("openvsx not migrated: %s", content)
	}
}

func TestMigrateAlreadyCurrent(t *testing.T) {
	dir := t.TempDir()
	writeSchemaVersion(dir, LatestSchemaVersion)

	applied, err := Migrate(dir)
	if err != nil {
		t.Fatal(err)
	}
	if applied != 0 {
		t.Errorf("expected 0 migrations applied, got %d", applied)
	}
}

func TestEnsureSchemaVersion(t *testing.T) {
	dir := t.TempDir()

	ensureSchemaVersion(dir)
	v, _ := SchemaVersion(dir)
	if v != LatestSchemaVersion {
		t.Errorf("expected version %d, got %d", LatestSchemaVersion, v)
	}

	// Should not overwrite existing version
	writeSchemaVersion(dir, 0)
	ensureSchemaVersion(dir)
	v, _ = SchemaVersion(dir)
	if v != 0 {
		t.Errorf("expected version 0 (not overwritten), got %d", v)
	}
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (len(s) >= len(substr)) && (s != "" && substr != "") && (indexOf(s, substr) >= 0)
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
