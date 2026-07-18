package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// writeStatusFixture lays down a config with one project whose github metric
// series last changed on 2025-01-01 but was last checked (watermarked) on a
// later date, plus an unfetched pypi source. It returns the config path.
func writeStatusFixture(t *testing.T, watermarkDate string) string {
	t.Helper()
	cfgPath := setupTestConfig(t, `[data]
dir = "data"

[projects.alpha]
name = "Alpha"
github = "org/alpha"
pypi = "alpha-pkg"
`)
	dir := filepath.Dir(cfgPath)

	metricsDir := filepath.Join(dir, "data", "metrics", "github", "alpha")
	if err := os.MkdirAll(metricsDir, 0755); err != nil {
		t.Fatal(err)
	}
	writeTempFile(t, metricsDir, "2025-01-01.jsonl",
		`{"source":"github","metric":"total_stars","project_id":"alpha","target":"org/alpha","date":"2025-01-01","value":100,"tags":{}}`+"\n")
	writeTempFile(t, metricsDir, "_watermark.json",
		`{"source":"github","project_id":"alpha","target":"org/alpha","date":"`+watermarkDate+`"}`+"\n")

	writeTempFile(t, filepath.Join(dir, "data"), ".schema-version", currentSchemaVersion)
	return cfgPath
}

func TestStatusReportsWatermarkPerTarget(t *testing.T) {
	cfgPath := writeStatusFixture(t, "2025-06-15")

	_, buf, err := execCmd(cfgPath, "status")
	if err != nil {
		t.Fatal(err)
	}

	output := buf.String()
	// The watermark date, not the last changed row (2025-01-01), is reported.
	assertContains(t, output, "org/alpha")
	assertContains(t, output, "2025-06-15")
	assertNotContains(t, output, "2025-01-01")
	// pypi was never fetched.
	assertContains(t, output, "never")
}

func TestStatusJSON(t *testing.T) {
	cfgPath := writeStatusFixture(t, "2025-06-15")

	_, buf, err := execCmd(cfgPath, "status", "--json")
	if err != nil {
		t.Fatal(err)
	}

	var rows []statusRow
	if err := json.Unmarshal(buf.Bytes(), &rows); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, buf.String())
	}

	var github, pypi *statusRow
	for i := range rows {
		switch rows[i].Source {
		case "github":
			github = &rows[i]
		case "pypi":
			pypi = &rows[i]
		}
	}
	if github == nil || pypi == nil {
		t.Fatalf("expected github and pypi rows, got %+v", rows)
	}
	if github.Target != "org/alpha" {
		t.Errorf("github target = %q, want org/alpha", github.Target)
	}
	if github.LastDate != "2025-06-15" {
		t.Errorf("github last_date = %q, want watermark 2025-06-15", github.LastDate)
	}
	if pypi.LastDate != "" || !pypi.Stale {
		t.Errorf("pypi should be never-fetched and stale, got %+v", *pypi)
	}
}

func TestStatusStaleOnly(t *testing.T) {
	// Recent watermark keeps github fresh, so --stale-only drops it.
	cfgPath := writeStatusFixture(t, "2025-06-15")

	_, buf, err := execCmd(cfgPath, "status", "--stale-only", "--stale-days", "100000")
	if err != nil {
		t.Fatal(err)
	}

	output := buf.String()
	assertNotContains(t, output, "org/alpha")
	assertContains(t, output, "pypi")
}

func TestStatusSingleProjectFilter(t *testing.T) {
	cfgPath := writeStatusFixture(t, "2025-06-15")

	if _, _, err := execCmd(cfgPath, "status", "nonexistent"); err == nil {
		t.Fatal("expected error for nonexistent project")
	}

	if _, buf, err := execCmd(cfgPath, "status", "alpha"); err != nil {
		t.Fatalf("status alpha: %v\n%s", err, buf.String())
	}
}
