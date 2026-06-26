package store

import (
	"path/filepath"
	"testing"

	"github.com/jeroenjanssens/velocirepo/internal/source"
)

func TestWriteYouTubeIndex(t *testing.T) {
	dir := t.TempDir()
	dataDir := filepath.Join(dir, "data")

	entries := []source.YouTubeIndexEntry{
		{VideoID: "vid1", Title: "First Video", PublishedAt: "2025-01-01T10:00:00Z", Channel: "@Test"},
		{VideoID: "vid2", Title: "Second Video", PublishedAt: "2025-02-01T10:00:00Z", Channel: "@Test"},
	}

	if err := WriteYouTubeIndex(dataDir, "my-proj", entries); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(dataDir, "metrics", "youtube", "my-proj", "index.jsonl")
	read, err := readYouTubeIndex(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(read) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(read))
	}
	if read[0].VideoID != "vid1" || read[0].Title != "First Video" {
		t.Errorf("unexpected entry 0: %+v", read[0])
	}
}

func TestWriteYouTubeIndexMerge(t *testing.T) {
	dir := t.TempDir()
	dataDir := filepath.Join(dir, "data")

	initial := []source.YouTubeIndexEntry{
		{VideoID: "vid1", Title: "Old Title", PublishedAt: "2025-01-01T10:00:00Z", Channel: "@Test"},
		{VideoID: "vid2", Title: "Video Two", PublishedAt: "2025-02-01T10:00:00Z", Channel: "@Test"},
	}
	if err := WriteYouTubeIndex(dataDir, "proj", initial); err != nil {
		t.Fatal(err)
	}

	update := []source.YouTubeIndexEntry{
		{VideoID: "vid1", Title: "New Title", PublishedAt: "2025-01-01T10:00:00Z", Channel: "@Test"},
		{VideoID: "vid3", Title: "Video Three", PublishedAt: "2025-03-01T10:00:00Z", Channel: "@Test"},
	}
	if err := WriteYouTubeIndex(dataDir, "proj", update); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(dataDir, "metrics", "youtube", "proj", "index.jsonl")
	read, err := readYouTubeIndex(path)
	if err != nil {
		t.Fatal(err)
	}

	if len(read) != 3 {
		t.Fatalf("expected 3 entries after merge, got %d", len(read))
	}
	// vid1 should have updated title
	if read[0].VideoID != "vid1" || read[0].Title != "New Title" {
		t.Errorf("expected vid1 with updated title, got %+v", read[0])
	}
	// vid2 preserved from initial
	if read[1].VideoID != "vid2" {
		t.Errorf("expected vid2 preserved, got %+v", read[1])
	}
	// vid3 appended
	if read[2].VideoID != "vid3" {
		t.Errorf("expected vid3 appended, got %+v", read[2])
	}
}

func TestYouTubeIndexDuckDBView(t *testing.T) {
	dir := t.TempDir()
	dataDir := filepath.Join(dir, "data")

	dur := int64(330)
	contentEntries := []source.ContentEntry{
		{Source: "youtube", Target: "@TestChan", ID: "abc123", Title: "Test Video", PublishedAt: "2025-06-01T10:00:00Z", Duration: &dur, Tags: []string{"go", "tutorial"}, Type: "video"},
	}
	if err := WriteContent(dataDir, "youtube", "proj", "videos.jsonl", contentEntries); err != nil {
		t.Fatal(err)
	}

	results, _, err := QueryLive(dataDir, nil, nil, "SELECT video_id, title, channel, duration FROM youtube_index")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 row, got %d", len(results))
	}
	if results[0]["video_id"] != "abc123" {
		t.Errorf("expected video_id=abc123, got %v", results[0]["video_id"])
	}
	if results[0]["title"] != "Test Video" {
		t.Errorf("expected title=Test Video, got %v", results[0]["title"])
	}
	if results[0]["duration"].(int64) != 330 {
		t.Errorf("expected duration=330, got %v", results[0]["duration"])
	}
}
