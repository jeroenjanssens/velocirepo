package store

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jeroenjanssens/velocirepo/internal/source"
)

func TestWriteAndReadRecords(t *testing.T) {
	dir := t.TempDir()
	records := []source.Record{
		{Metric: "downloads", ProjectID: "mylib", Date: "2025-06-01", Value: 100},
		{Metric: "downloads", ProjectID: "mylib", Date: "2025-06-01", Value: 200},
		{Metric: "downloads", ProjectID: "mylib", Date: "2025-06-02", Value: 150},
	}

	if err := WriteRecords(dir, "pypi", "mylib", records); err != nil {
		t.Fatalf("WriteRecords failed: %v", err)
	}

	// Check files exist
	path1 := filepath.Join(dir, "pypi", "mylib", "2025-06-01.jsonl")
	path2 := filepath.Join(dir, "pypi", "mylib", "2025-06-02.jsonl")

	if _, err := os.Stat(path1); err != nil {
		t.Fatalf("file not created: %s", path1)
	}
	if _, err := os.Stat(path2); err != nil {
		t.Fatalf("file not created: %s", path2)
	}

	// Read back day 1
	got, err := ReadRecords(path1)
	if err != nil {
		t.Fatalf("ReadRecords failed: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d records, want 2", len(got))
	}
	if got[0].Value != 100 {
		t.Errorf("got[0].Value = %d, want 100", got[0].Value)
	}
	if got[1].Value != 200 {
		t.Errorf("got[1].Value = %d, want 200", got[1].Value)
	}

	// Read back day 2
	got, err = ReadRecords(path2)
	if err != nil {
		t.Fatalf("ReadRecords failed: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d records, want 1", len(got))
	}
}

func TestWriteRecordsWithTags(t *testing.T) {
	dir := t.TempDir()
	records := []source.Record{
		{Metric: "views", ProjectID: "mylib", Date: "2025-06-01", Value: 500, Tags: map[string]string{"video_id": "abc123"}},
	}

	if err := WriteRecords(dir, "youtube", "mylib", records); err != nil {
		t.Fatalf("WriteRecords failed: %v", err)
	}

	path := filepath.Join(dir, "youtube", "mylib", "2025-06-01.jsonl")
	got, err := ReadRecords(path)
	if err != nil {
		t.Fatalf("ReadRecords failed: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d records, want 1", len(got))
	}
	if got[0].Tags["video_id"] != "abc123" {
		t.Errorf("Tags[video_id] = %q, want %q", got[0].Tags["video_id"], "abc123")
	}
}

func TestLastDateDaily(t *testing.T) {
	dir := t.TempDir()
	projDir := filepath.Join(dir, "github", "myproject")
	os.MkdirAll(projDir, 0755)

	for _, name := range []string{"2025-01-15.jsonl", "2025-03-20.jsonl", "2025-02-10.jsonl"} {
		os.WriteFile(filepath.Join(projDir, name), []byte(`{}`+"\n"), 0644)
	}

	got, err := LastDate(dir, "github", "myproject")
	if err != nil {
		t.Fatalf("LastDate failed: %v", err)
	}

	want := time.Date(2025, 3, 20, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("LastDate = %v, want %v", got, want)
	}
}

func TestLastDateMonthly(t *testing.T) {
	dir := t.TempDir()
	projDir := filepath.Join(dir, "pypi", "mylib")
	os.MkdirAll(projDir, 0755)

	for _, name := range []string{"2025-01.jsonl", "2025-03.jsonl"} {
		os.WriteFile(filepath.Join(projDir, name), []byte(`{}`+"\n"), 0644)
	}

	got, err := LastDate(dir, "pypi", "mylib")
	if err != nil {
		t.Fatalf("LastDate failed: %v", err)
	}

	want := time.Date(2025, 3, 31, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("LastDate = %v, want %v", got, want)
	}
}

func TestLastDateYearly(t *testing.T) {
	dir := t.TempDir()
	projDir := filepath.Join(dir, "cran", "pkg")
	os.MkdirAll(projDir, 0755)

	os.WriteFile(filepath.Join(projDir, "2024.jsonl"), []byte(`{}`+"\n"), 0644)

	got, err := LastDate(dir, "cran", "pkg")
	if err != nil {
		t.Fatalf("LastDate failed: %v", err)
	}

	want := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("LastDate = %v, want %v", got, want)
	}
}

func TestLastDateEmpty(t *testing.T) {
	dir := t.TempDir()

	got, err := LastDate(dir, "github", "nonexistent")
	if err != nil {
		t.Fatalf("LastDate failed: %v", err)
	}

	if !got.IsZero() {
		t.Errorf("LastDate = %v, want zero time", got)
	}
}

func TestLastDateMixed(t *testing.T) {
	dir := t.TempDir()
	projDir := filepath.Join(dir, "github", "proj")
	os.MkdirAll(projDir, 0755)

	for _, name := range []string{"2024.jsonl", "2025-01.jsonl", "2025-02-15.jsonl"} {
		os.WriteFile(filepath.Join(projDir, name), []byte(`{}`+"\n"), 0644)
	}

	got, err := LastDate(dir, "github", "proj")
	if err != nil {
		t.Fatalf("LastDate failed: %v", err)
	}

	want := time.Date(2025, 2, 15, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("LastDate = %v, want %v", got, want)
	}
}
