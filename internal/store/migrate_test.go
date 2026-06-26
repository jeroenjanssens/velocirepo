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
	if applied != 5 {
		t.Errorf("expected 5 migrations applied, got %d", applied)
	}

	v, _ := SchemaVersion(dir)
	if v != 5 {
		t.Errorf("expected schema version 5, got %d", v)
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

func TestMigrate4to5YouTubeIndex(t *testing.T) {
	dir := t.TempDir()
	writeSchemaVersion(dir, 4)

	// Set up a YouTube index in the old location
	youtubeDir := filepath.Join(dir, "metrics", "youtube", "my-proj")
	os.MkdirAll(youtubeDir, 0755)
	os.WriteFile(filepath.Join(youtubeDir, "index.jsonl"),
		[]byte(`{"video_id":"abc123","title":"Test Video","published_at":"2025-06-01T10:00:00Z","channel":"@TestChan","duration":330,"tags":["go","tutorial"]}`+"\n"+
			`{"video_id":"def456","title":"Second Video","published_at":"2025-07-01T10:00:00Z","channel":"@TestChan","duration":600}`+"\n"), 0644)

	applied, err := Migrate(dir)
	if err != nil {
		t.Fatal(err)
	}
	if applied != 1 {
		t.Errorf("expected 1 migration applied, got %d", applied)
	}

	// Old file should be removed
	if _, err := os.Stat(filepath.Join(youtubeDir, "index.jsonl")); !os.IsNotExist(err) {
		t.Error("old index.jsonl should have been removed")
	}

	// New file should exist
	newPath := filepath.Join(dir, "content", "youtube", "my-proj", "videos.jsonl")
	data, err := os.ReadFile(newPath)
	if err != nil {
		t.Fatalf("new content file not created: %v", err)
	}

	content := string(data)
	if !contains(content, `"source":"youtube"`) {
		t.Error("expected source=youtube in migrated content")
	}
	if !contains(content, `"id":"abc123"`) {
		t.Error("expected id=abc123 in migrated content")
	}
	if !contains(content, `"target":"@TestChan"`) {
		t.Error("expected target=@TestChan in migrated content")
	}
	if !contains(content, `"url":"https://youtube.com/watch?v=abc123"`) {
		t.Error("expected url for abc123 in migrated content")
	}
	if !contains(content, `"type":"video"`) {
		t.Error("expected type=video in migrated content")
	}
	if !contains(content, `"duration":330`) {
		t.Error("expected duration=330 in migrated content")
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
