package store

import (
	"strings"
	"testing"

	"github.com/jeroenjanssens/velocirepo/internal/source"
)

func TestRewriteProjectID(t *testing.T) {
	dir := t.TempDir()
	metricPath := metricsPath(dir, "pypi", "new-proj", "2026-06-01")
	eventPath := eventsPath(dir, "github", "new-proj", "2026-06-01")
	watermarkPath := metricWatermarkPath(dir, "pypi", "new-proj", "2026-06-01")
	writeTestRecords(t, metricPath, source.Record{Source: "pypi", Metric: "daily_downloads", ProjectID: "old-proj", Target: "pkg", Date: "2026-06-01", Value: 1})
	writeTestEvents(t, eventPath, source.Event{Source: "github", Type: "star", ProjectID: "old-proj", Target: "org/repo", Datetime: "2026-06-01T00:00:00Z"})
	writeTestRaw(t, watermarkPath, []string{`{"source":"pypi","metric":"total_downloads","project_id":"old-proj","target":"pkg","date":"2026-06-01"}`})

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

	watermark := readRawLines(t, watermarkPath)[0]
	if !strings.Contains(watermark, `"project_id":"new-proj"`) {
		t.Fatalf("watermark project_id was not rewritten: %s", watermark)
	}
}
