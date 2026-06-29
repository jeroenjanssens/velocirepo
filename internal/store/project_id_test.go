package store

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jeroenjanssens/velocirepo/internal/source"
)

func TestRewriteProjectID(t *testing.T) {
	dir := t.TempDir()
	metricsDir := filepath.Join(dir, "metrics", "pypi", "new-proj")
	eventsDir := filepath.Join(dir, "events", "github", "new-proj")
	if err := os.MkdirAll(metricsDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(eventsDir, 0755); err != nil {
		t.Fatal(err)
	}

	metricPath := filepath.Join(metricsDir, "2026-06-01.jsonl")
	eventPath := filepath.Join(eventsDir, "2026-06-01.jsonl")
	if err := writeFileAtomic(metricPath, []source.Record{{Source: "pypi", Metric: "daily_downloads", ProjectID: "old-proj", Target: "pkg", Date: "2026-06-01", Value: 1}}); err != nil {
		t.Fatal(err)
	}
	if err := writeEventsFileAtomic(eventPath, []source.Event{{Source: "github", Type: "star", ProjectID: "old-proj", Target: "org/repo", Datetime: "2026-06-01T00:00:00Z"}}); err != nil {
		t.Fatal(err)
	}

	if err := RewriteProjectID(dir, "old-proj", "new-proj"); err != nil {
		t.Fatalf("RewriteProjectID failed: %v", err)
	}

	metrics, err := ReadRecords(metricPath)
	if err != nil {
		t.Fatal(err)
	}
	if metrics[0].ProjectID != "new-proj" {
		t.Fatalf("metric ProjectID = %q, want new-proj", metrics[0].ProjectID)
	}

	events, err := ReadEvents(eventPath)
	if err != nil {
		t.Fatal(err)
	}
	if events[0].ProjectID != "new-proj" {
		t.Fatalf("event ProjectID = %q, want new-proj", events[0].ProjectID)
	}
}
