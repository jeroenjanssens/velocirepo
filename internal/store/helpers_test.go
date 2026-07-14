package store

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jeroenjanssens/velocirepo/internal/source"
)

func testDataDir(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "data")
}

func metricsPath(dataDir, sourceName, project, date string) string {
	return filepath.Join(dataDir, "metrics", sourceName, project, jsonlName(date))
}

func eventsPath(dataDir, sourceName, project, date string) string {
	return filepath.Join(dataDir, "events", sourceName, project, jsonlName(date))
}

func metricWatermarkPath(dataDir, sourceName, project string) string {
	return filepath.Join(dataDir, "metrics", sourceName, project, WatermarkFileName)
}

func contentPath(dataDir, sourceName, project, name string) string {
	return filepath.Join(dataDir, "content", sourceName, project, jsonlName(name))
}

func jsonlName(name string) string {
	if strings.HasSuffix(name, ".jsonl") {
		return name
	}
	return name + ".jsonl"
}

func writeTestRecords(t *testing.T, path string, records ...source.Record) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := writeFileAtomic(path, records); err != nil {
		t.Fatal(err)
	}
}

func writeTestEvents(t *testing.T, path string, events ...source.Event) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := writeEventsFileAtomic(path, events); err != nil {
		t.Fatal(err)
	}
}

func writeTestRaw(t *testing.T, path string, lines []string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	for _, line := range lines {
		if _, err := f.WriteString(line + "\n"); err != nil {
			t.Fatal(err)
		}
	}
}

func readRawLines(t *testing.T, path string) []string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var lines []string
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

func assertIssueTypes(t *testing.T, got []Issue, want ...IssueType) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("expected %d issues, got %d: %v", len(want), len(got), got)
	}
	for i, wantType := range want {
		if got[i].Type != wantType {
			t.Errorf("issue %d type = %v, want %v", i, got[i].Type, wantType)
		}
	}
}
