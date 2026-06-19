package store

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jeroenjanssens/velocirepo/internal/source"
)

func TestQueryLive(t *testing.T) {
	dir := t.TempDir()
	dataDir := filepath.Join(dir, "data")

	records := []source.Record{
		{Metric: "stars", ProjectID: "my-proj", Date: "2025-06-01", Value: 10},
		{Metric: "forks", ProjectID: "my-proj", Date: "2025-06-01", Value: 3},
		{Metric: "stars", ProjectID: "my-proj", Date: "2025-06-02", Value: 15},
	}
	if err := WriteRecords(dataDir, "cran", "my-proj", records); err != nil {
		t.Fatal(err)
	}

	pypiRecords := []source.Record{
		{Metric: "downloads", ProjectID: "my-proj", Date: "2025-06-01", Value: 500, Tags: map[string]string{"version": "1.0.0"}},
	}
	if err := WriteRecords(dataDir, "pypi", "my-proj", pypiRecords); err != nil {
		t.Fatal(err)
	}

	results, _, err := QueryLive(dataDir, nil, "SELECT COUNT(*) AS cnt FROM metrics")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 row, got %d", len(results))
	}
	cnt, ok := results[0]["cnt"].(int64)
	if !ok {
		t.Fatalf("unexpected count type: %T = %v", results[0]["cnt"], results[0]["cnt"])
	}
	if cnt != 4 {
		t.Fatalf("expected 4 rows, got %d", cnt)
	}

	results, _, err = QueryLive(dataDir, nil, "SELECT DISTINCT source FROM metrics ORDER BY source")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 sources, got %d", len(results))
	}
	if results[0]["source"] != "cran" {
		t.Errorf("expected first source=cran, got %v", results[0]["source"])
	}
	if results[1]["source"] != "pypi" {
		t.Errorf("expected second source=pypi, got %v", results[1]["source"])
	}

	results, _, err = QueryLive(dataDir, nil, "SELECT metric, value FROM metrics WHERE source = 'pypi'")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 pypi row, got %d", len(results))
	}
	if results[0]["value"].(int64) != 500 {
		t.Errorf("expected value=500, got %v", results[0]["value"])
	}
}

func TestQueryLiveAggregation(t *testing.T) {
	dir := t.TempDir()
	dataDir := filepath.Join(dir, "data")

	records := []source.Record{
		{Metric: "stars", ProjectID: "proj-a", Date: "2025-01-01", Value: 100},
		{Metric: "stars", ProjectID: "proj-a", Date: "2025-01-02", Value: 105},
		{Metric: "forks", ProjectID: "proj-a", Date: "2025-01-01", Value: 20},
		{Metric: "stars", ProjectID: "proj-b", Date: "2025-01-01", Value: 50},
	}
	if err := WriteRecords(dataDir, "pypi", "proj-a", records[:3]); err != nil {
		t.Fatal(err)
	}
	if err := WriteRecords(dataDir, "pypi", "proj-b", records[3:]); err != nil {
		t.Fatal(err)
	}

	results, _, err := QueryLive(dataDir, nil, "SELECT project, SUM(value) AS total FROM metrics WHERE metric = 'stars' GROUP BY project ORDER BY project")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(results))
	}
	if results[0]["project"] != "proj-a" {
		t.Errorf("expected proj-a, got %v", results[0]["project"])
	}
}

func TestQueryLiveEmptyDir(t *testing.T) {
	dir := t.TempDir()
	dataDir := filepath.Join(dir, "data")
	os.MkdirAll(dataDir, 0755)

	results, _, err := QueryLive(dataDir, nil, "SELECT COUNT(*) AS cnt FROM metrics")
	if err != nil {
		t.Fatal(err)
	}
	cnt := results[0]["cnt"].(int64)
	if cnt != 0 {
		t.Fatalf("expected 0 rows, got %d", cnt)
	}
}

func TestQueryLiveInvalidSQL(t *testing.T) {
	dir := t.TempDir()
	dataDir := filepath.Join(dir, "data")
	os.MkdirAll(dataDir, 0755)

	_, _, err := QueryLive(dataDir, nil, "SELECT * FROM nonexistent_table")
	if err == nil {
		t.Fatal("expected error for invalid query")
	}
}

func TestSchemaLive(t *testing.T) {
	dir := t.TempDir()
	dataDir := filepath.Join(dir, "data")

	records := []source.Record{
		{Metric: "stars", ProjectID: "test", Date: "2025-01-01", Value: 1},
	}
	if err := WriteRecords(dataDir, "pypi", "test", records); err != nil {
		t.Fatal(err)
	}

	events := []source.GitHubEvent{
		{EventType: "star", ProjectID: "test", GitHubRepo: "owner/repo", Datetime: "2025-01-01T10:00:00Z", User: "alice"},
	}
	if err := WriteGitHubEvents(dataDir, "github", "test", events); err != nil {
		t.Fatal(err)
	}

	cols, err := SchemaLive(dataDir, nil)
	if err != nil {
		t.Fatal(err)
	}

	eventsExpected := []string{"project", "source", "event_type", "github_repo", "datetime", "user"}
	metricsExpected := []string{"project", "source", "target", "metric", "date", "value", "tags"}
	projectsExpected := []string{"id", "name", "description", "color", "tags", "website", "logo"}

	var eventsCols, metricsCols, projectsCols []SchemaColumn
	for _, c := range cols {
		switch c.Table {
		case "github_events":
			eventsCols = append(eventsCols, c)
		case "metrics":
			metricsCols = append(metricsCols, c)
		case "projects":
			projectsCols = append(projectsCols, c)
		}
	}

	if len(eventsCols) != 6 {
		t.Fatalf("expected 6 github_events columns, got %d", len(eventsCols))
	}
	if len(metricsCols) != 7 {
		t.Fatalf("expected 7 metrics columns, got %d", len(metricsCols))
	}
	if len(projectsCols) != 7 {
		t.Fatalf("expected 7 projects columns, got %d", len(projectsCols))
	}

	for i, exp := range eventsExpected {
		if eventsCols[i].Column != exp {
			t.Errorf("github_events column %d: expected %s, got %s", i, exp, eventsCols[i].Column)
		}
	}

	for i, exp := range metricsExpected {
		if metricsCols[i].Column != exp {
			t.Errorf("metrics column %d: expected %s, got %s", i, exp, metricsCols[i].Column)
		}
	}

	for i, exp := range projectsExpected {
		if projectsCols[i].Column != exp {
			t.Errorf("projects column %d: expected %s, got %s", i, exp, projectsCols[i].Column)
		}
	}
}

func TestQueryLiveGitHubEvents(t *testing.T) {
	dir := t.TempDir()
	dataDir := filepath.Join(dir, "data")

	events := []source.GitHubEvent{
		{EventType: "star", ProjectID: "my-proj", GitHubRepo: "owner/repo", Datetime: "2025-06-01T10:00:00Z", User: "alice"},
		{EventType: "fork", ProjectID: "my-proj", GitHubRepo: "owner/repo", Datetime: "2025-06-01T11:00:00Z", User: "bob"},
		{EventType: "issue_open", ProjectID: "my-proj", GitHubRepo: "owner/repo", Datetime: "2025-06-02T09:00:00Z", User: "carol"},
	}
	if err := WriteGitHubEvents(dataDir, "github", "my-proj", events); err != nil {
		t.Fatal(err)
	}

	results, _, err := QueryLive(dataDir, nil, "SELECT COUNT(*) AS cnt FROM github_events")
	if err != nil {
		t.Fatal(err)
	}
	cnt := results[0]["cnt"].(int64)
	if cnt != 3 {
		t.Fatalf("expected 3 events, got %d", cnt)
	}

	results, _, err = QueryLive(dataDir, nil, "SELECT event_type, \"user\" FROM github_events WHERE event_type = 'star'")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 star event, got %d", len(results))
	}
	if results[0]["user"] != "alice" {
		t.Errorf("expected user=alice, got %v", results[0]["user"])
	}
}

func TestMetricsViewIncludesGitHubAggregated(t *testing.T) {
	dir := t.TempDir()
	dataDir := filepath.Join(dir, "data")

	records := []source.Record{
		{Metric: "downloads", ProjectID: "test", Date: "2025-06-01", Value: 10},
	}
	if err := WriteRecords(dataDir, "pypi", "test", records); err != nil {
		t.Fatal(err)
	}

	events := []source.GitHubEvent{
		{EventType: "star", ProjectID: "test", GitHubRepo: "owner/repo", Datetime: "2025-06-01T10:00:00Z", User: "alice"},
	}
	if err := WriteGitHubEvents(dataDir, "github", "test", events); err != nil {
		t.Fatal(err)
	}

	results, _, err := QueryLive(dataDir, nil, "SELECT COUNT(*) AS cnt FROM metrics")
	if err != nil {
		t.Fatal(err)
	}
	cnt := results[0]["cnt"].(int64)
	if cnt != 2 {
		t.Fatalf("expected 2 metric rows (1 pypi + 1 aggregated github event), got %d", cnt)
	}
}

func TestQueryLiveGitHubView(t *testing.T) {
	dir := t.TempDir()
	dataDir := filepath.Join(dir, "data")

	events := []source.GitHubEvent{
		{EventType: "star", ProjectID: "my-proj", GitHubRepo: "owner/repo", Datetime: "2025-06-01T10:00:00Z", User: "alice"},
		{EventType: "star", ProjectID: "my-proj", GitHubRepo: "owner/repo", Datetime: "2025-06-01T12:00:00Z", User: "bob"},
		{EventType: "fork", ProjectID: "my-proj", GitHubRepo: "owner/repo", Datetime: "2025-06-01T11:00:00Z", User: "carol"},
	}
	if err := WriteGitHubEvents(dataDir, "github", "my-proj", events); err != nil {
		t.Fatal(err)
	}

	results, _, err := QueryLive(dataDir, nil,
		"SELECT metric, value FROM metrics WHERE project = 'my-proj' AND source = 'github' AND date = '2025-06-01' ORDER BY metric")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(results))
	}
	if results[0]["metric"] != "fork" || results[0]["value"].(int64) != 1 {
		t.Errorf("unexpected fork row: %v", results[0])
	}
	if results[1]["metric"] != "star" || results[1]["value"].(int64) != 2 {
		t.Errorf("unexpected star row: %v", results[1])
	}
}

