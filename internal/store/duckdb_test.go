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
	if err := WriteRecords(dataDir, "github", "my-proj", records); err != nil {
		t.Fatal(err)
	}

	pypiRecords := []source.Record{
		{Metric: "downloads", ProjectID: "my-proj", Date: "2025-06-01", Value: 500, Tags: map[string]string{"version": "1.0.0"}},
	}
	if err := WriteRecords(dataDir, "pypi", "my-proj", pypiRecords); err != nil {
		t.Fatal(err)
	}

	results, err := QueryLive(dataDir, "SELECT COUNT(*) AS cnt FROM metrics")
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

	results, err = QueryLive(dataDir, "SELECT DISTINCT source FROM metrics ORDER BY source")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 sources, got %d", len(results))
	}
	if results[0]["source"] != "github" {
		t.Errorf("expected first source=github, got %v", results[0]["source"])
	}
	if results[1]["source"] != "pypi" {
		t.Errorf("expected second source=pypi, got %v", results[1]["source"])
	}

	results, err = QueryLive(dataDir, "SELECT metric, value FROM metrics WHERE source = 'pypi'")
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
	if err := WriteRecords(dataDir, "github", "proj-a", records[:3]); err != nil {
		t.Fatal(err)
	}
	if err := WriteRecords(dataDir, "github", "proj-b", records[3:]); err != nil {
		t.Fatal(err)
	}

	results, err := QueryLive(dataDir, "SELECT project, SUM(value) AS total FROM metrics WHERE metric = 'stars' GROUP BY project ORDER BY project")
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

	results, err := QueryLive(dataDir, "SELECT COUNT(*) AS cnt FROM metrics")
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

	_, err := QueryLive(dataDir, "SELECT * FROM nonexistent_table")
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
	if err := WriteRecords(dataDir, "github", "test", records); err != nil {
		t.Fatal(err)
	}

	cols, err := SchemaLive(dataDir)
	if err != nil {
		t.Fatal(err)
	}

	if len(cols) != 6 {
		t.Fatalf("expected 6 columns, got %d", len(cols))
	}

	expected := []string{"project", "source", "metric", "date", "value", "tags"}
	for i, exp := range expected {
		if cols[i].Column != exp {
			t.Errorf("column %d: expected %s, got %s", i, exp, cols[i].Column)
		}
	}
}
