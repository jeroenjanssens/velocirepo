package store

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jeroenjanssens/velocirepo/internal/source"
)

func ContentDir(dataDir, sourceName, projectID string) string {
	return filepath.Join(dataDir, "content", sourceName, projectID)
}

func WriteContent(dataDir, sourceName, projectID, filename string, entries []source.ContentEntry) error {
	dir := ContentDir(dataDir, sourceName, projectID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create directory %s: %w", dir, err)
	}

	path := filepath.Join(dir, filename)

	existing, err := ReadContent(path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("read existing content: %w", err)
	}

	merged := mergeContentEntries(existing, entries)

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
			return fmt.Errorf("marshal content entry: %w", err)
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

	ensureSchemaVersion(dataDir)
	return nil
}

func ReadContent(path string) ([]source.ContentEntry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var entries []source.ContentEntry
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		var e source.ContentEntry
		if err := json.Unmarshal(scanner.Bytes(), &e); err != nil {
			continue
		}
		entries = append(entries, e)
	}
	return entries, scanner.Err()
}

func mergeContentEntries(existing, incoming []source.ContentEntry) []source.ContentEntry {
	byID := make(map[string]source.ContentEntry, len(existing))
	var order []string

	for _, e := range existing {
		byID[e.ID] = e
		order = append(order, e.ID)
	}

	for _, e := range incoming {
		if _, exists := byID[e.ID]; !exists {
			order = append(order, e.ID)
		}
		byID[e.ID] = e
	}

	result := make([]source.ContentEntry, 0, len(order))
	for _, id := range order {
		result = append(result, byID[id])
	}
	return result
}
