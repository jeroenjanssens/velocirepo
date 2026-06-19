package store

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jeroenjanssens/velocirepo/internal/source"
)

func WriteYouTubeIndex(dataDir, projectID string, entries []source.YouTubeIndexEntry) error {
	dir := filepath.Join(dataDir, "youtube", projectID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create directory %s: %w", dir, err)
	}

	path := filepath.Join(dir, "index.jsonl")

	existing, err := readYouTubeIndex(path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("read existing index: %w", err)
	}

	merged := mergeIndexEntries(existing, entries)

	tmp, err := os.CreateTemp(dir, ".tmp-*.jsonl")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmp.Name()

	w := bufio.NewWriter(tmp)
	for _, e := range merged {
		data, err := json.Marshal(e)
		if err != nil {
			tmp.Close()
			os.Remove(tmpPath)
			return fmt.Errorf("marshal index entry: %w", err)
		}
		w.Write(data)
		w.WriteByte('\n')
	}

	if err := w.Flush(); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return err
	}
	return nil
}

func readYouTubeIndex(path string) ([]source.YouTubeIndexEntry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var entries []source.YouTubeIndexEntry
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		var e source.YouTubeIndexEntry
		if err := json.Unmarshal(scanner.Bytes(), &e); err != nil {
			continue
		}
		entries = append(entries, e)
	}
	return entries, scanner.Err()
}

func mergeIndexEntries(existing, incoming []source.YouTubeIndexEntry) []source.YouTubeIndexEntry {
	byID := make(map[string]source.YouTubeIndexEntry, len(existing))
	var order []string

	for _, e := range existing {
		byID[e.VideoID] = e
		order = append(order, e.VideoID)
	}

	for _, e := range incoming {
		if _, exists := byID[e.VideoID]; !exists {
			order = append(order, e.VideoID)
		}
		byID[e.VideoID] = e
	}

	result := make([]source.YouTubeIndexEntry, 0, len(order))
	for _, id := range order {
		result = append(result, byID[id])
	}
	return result
}
