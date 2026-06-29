package store

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// RewriteProjectID rewrites project_id fields in JSONL files below root.
func RewriteProjectID(root, oldID, newID string) error {
	if oldID == "" || oldID == newID {
		return nil
	}

	return filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".jsonl") {
			return nil
		}
		if err := rewriteProjectIDFile(path, oldID, newID); err != nil {
			return fmt.Errorf("rewrite %s: %w", path, err)
		}
		return nil
	})
}

func rewriteProjectIDFile(path, oldID, newID string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	encodedNewID, err := json.Marshal(newID)
	if err != nil {
		return err
	}

	var lines [][]byte
	changed := false
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Bytes()
		var record map[string]json.RawMessage
		if err := json.Unmarshal(line, &record); err != nil {
			return err
		}

		if raw, ok := record["project_id"]; ok {
			var projectID string
			if err := json.Unmarshal(raw, &projectID); err != nil {
				return fmt.Errorf("project_id is not a string: %w", err)
			}
			if projectID == oldID {
				record["project_id"] = encodedNewID
				rewritten, err := json.Marshal(record)
				if err != nil {
					return err
				}
				lines = append(lines, rewritten)
				changed = true
				continue
			}
		}

		lines = append(lines, append([]byte(nil), line...))
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	if !changed {
		return nil
	}

	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".tmp-*.jsonl")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()

	w := bufio.NewWriter(tmp)
	for _, line := range lines {
		if _, err := w.Write(line); err != nil {
			tmp.Close()
			os.Remove(tmpPath)
			return err
		}
		if err := w.WriteByte('\n'); err != nil {
			tmp.Close()
			os.Remove(tmpPath)
			return err
		}
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
