package store

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jeroenjanssens/velocirepo/internal/source"
)

func TestExportParquet(t *testing.T) {
	dir := t.TempDir()
	dataDir := filepath.Join(dir, "data")

	records := []source.Record{
		{Metric: "stars", ProjectID: "my-proj", Date: "2025-06-01", Value: 10},
		{Metric: "forks", ProjectID: "my-proj", Date: "2025-06-01", Value: 3},
	}
	if err := WriteRecords(dataDir, "github", "my-proj", records); err != nil {
		t.Fatal(err)
	}

	outPath := filepath.Join(dir, "metrics.parquet")
	if err := ExportParquet(dataDir, outPath); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(outPath)
	if err != nil {
		t.Fatal("parquet file not created")
	}
	if info.Size() == 0 {
		t.Fatal("parquet file is empty")
	}

	// Verify we can read back the exported data
	results, err := QueryLive(dataDir, nil, "SELECT COUNT(*) AS cnt FROM '"+outPath+"'")
	if err != nil {
		t.Fatal(err)
	}
	cnt := results[0]["cnt"].(int64)
	if cnt != 2 {
		t.Fatalf("expected 2 rows in parquet, got %d", cnt)
	}
}

func TestExportCSV(t *testing.T) {
	dir := t.TempDir()
	dataDir := filepath.Join(dir, "data")

	records := []source.Record{
		{Metric: "downloads", ProjectID: "pkg", Date: "2025-06-01", Value: 500, Tags: map[string]string{"version": "1.0"}},
		{Metric: "downloads", ProjectID: "pkg", Date: "2025-06-02", Value: 600},
	}
	if err := WriteRecords(dataDir, "pypi", "pkg", records); err != nil {
		t.Fatal(err)
	}

	outPath := filepath.Join(dir, "metrics.csv")
	if err := ExportCSV(dataDir, outPath); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatal("csv file not created")
	}
	content := string(data)

	if len(content) == 0 {
		t.Fatal("csv file is empty")
	}
	// Should have a header line
	if content[:7] != "project" {
		t.Errorf("expected CSV to start with 'project' header, got %q", content[:20])
	}
}

func TestExportEmptyData(t *testing.T) {
	dir := t.TempDir()
	dataDir := filepath.Join(dir, "data")
	os.MkdirAll(dataDir, 0755)

	outPath := filepath.Join(dir, "metrics.parquet")
	if err := ExportParquet(dataDir, outPath); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(outPath)
	if err != nil {
		t.Fatal("parquet file not created for empty data")
	}
	if info.Size() == 0 {
		t.Fatal("parquet file is empty")
	}
}
