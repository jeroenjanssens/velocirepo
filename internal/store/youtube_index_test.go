package store

import (
	"path/filepath"
	"testing"

	"github.com/jeroenjanssens/velocirepo/internal/source"
)

func TestWriteContent(t *testing.T) {
	dir := t.TempDir()
	dataDir := filepath.Join(dir, "data")

	dur1 := int64(330)
	dur2 := int64(600)
	entries := []source.ContentEntry{
		{Source: "youtube", Target: "@Test", ID: "vid1", Title: "First Video", PublishedAt: "2025-01-01T10:00:00Z", Duration: &dur1, Type: "video"},
		{Source: "youtube", Target: "@Test", ID: "vid2", Title: "Second Video", PublishedAt: "2025-02-01T10:00:00Z", Duration: &dur2, Type: "video"},
	}

	if err := WriteContent(dataDir, "youtube", "my-proj", "videos.jsonl", entries); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(dataDir, "content", "youtube", "my-proj", "videos.jsonl")
	read, err := ReadContent(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(read) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(read))
	}
	if read[0].ID != "vid1" || read[0].Title != "First Video" {
		t.Errorf("unexpected entry 0: %+v", read[0])
	}
}

func TestWriteContentMerge(t *testing.T) {
	dir := t.TempDir()
	dataDir := filepath.Join(dir, "data")

	dur := int64(60)
	initial := []source.ContentEntry{
		{Source: "youtube", Target: "@Test", ID: "vid1", Title: "Old Title", PublishedAt: "2025-01-01T10:00:00Z", Duration: &dur, Type: "video"},
		{Source: "youtube", Target: "@Test", ID: "vid2", Title: "Video Two", PublishedAt: "2025-02-01T10:00:00Z", Duration: &dur, Type: "video"},
	}
	if err := WriteContent(dataDir, "youtube", "proj", "videos.jsonl", initial); err != nil {
		t.Fatal(err)
	}

	update := []source.ContentEntry{
		{Source: "youtube", Target: "@Test", ID: "vid1", Title: "New Title", PublishedAt: "2025-01-01T10:00:00Z", Duration: &dur, Type: "video"},
		{Source: "youtube", Target: "@Test", ID: "vid3", Title: "Video Three", PublishedAt: "2025-03-01T10:00:00Z", Duration: &dur, Type: "video"},
	}
	if err := WriteContent(dataDir, "youtube", "proj", "videos.jsonl", update); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(dataDir, "content", "youtube", "proj", "videos.jsonl")
	read, err := ReadContent(path)
	if err != nil {
		t.Fatal(err)
	}

	if len(read) != 3 {
		t.Fatalf("expected 3 entries after merge, got %d", len(read))
	}
	if read[0].ID != "vid1" || read[0].Title != "New Title" {
		t.Errorf("expected vid1 with updated title, got %+v", read[0])
	}
	if read[1].ID != "vid2" {
		t.Errorf("expected vid2 preserved, got %+v", read[1])
	}
	if read[2].ID != "vid3" {
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

func TestContentDuckDBView(t *testing.T) {
	dir := t.TempDir()
	dataDir := filepath.Join(dir, "data")

	dur := int64(330)
	entries := []source.ContentEntry{
		{Source: "youtube", Target: "@TestChan", ID: "abc123", Title: "Test Video", PublishedAt: "2025-06-01T10:00:00Z", Duration: &dur, Tags: []string{"go"}, Type: "video"},
	}
	if err := WriteContent(dataDir, "youtube", "proj", "videos.jsonl", entries); err != nil {
		t.Fatal(err)
	}

	results, _, err := QueryLive(dataDir, nil, nil, "SELECT source, target, id, title, type FROM content")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 row, got %d", len(results))
	}
	if results[0]["source"] != "youtube" {
		t.Errorf("expected source=youtube, got %v", results[0]["source"])
	}
	if results[0]["target"] != "@TestChan" {
		t.Errorf("expected target=@TestChan, got %v", results[0]["target"])
	}
	if results[0]["id"] != "abc123" {
		t.Errorf("expected id=abc123, got %v", results[0]["id"])
	}
	if results[0]["type"] != "video" {
		t.Errorf("expected type=video, got %v", results[0]["type"])
	}
}
