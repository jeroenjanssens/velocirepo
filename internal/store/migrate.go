package store

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const LatestSchemaVersion = 3

const schemaVersionFile = ".schema-version"

func SchemaVersion(dataDir string) (int, error) {
	path := filepath.Join(dataDir, schemaVersionFile)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, fmt.Errorf("read schema version: %w", err)
	}
	v, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0, fmt.Errorf("parse schema version: %w", err)
	}
	return v, nil
}

func writeSchemaVersion(dataDir string, version int) error {
	path := filepath.Join(dataDir, schemaVersionFile)
	return os.WriteFile(path, []byte(strconv.Itoa(version)+"\n"), 0644)
}

func ensureSchemaVersion(dataDir string) {
	path := filepath.Join(dataDir, schemaVersionFile)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		writeSchemaVersion(dataDir, LatestSchemaVersion)
	}
}

func CheckSchemaVersion(dataDir string) error {
	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		return nil
	}
	v, err := SchemaVersion(dataDir)
	if err != nil {
		return err
	}
	if v < LatestSchemaVersion {
		return fmt.Errorf("data schema is at version %d, but version %d is required; run `velocirepo migrate` to update", v, LatestSchemaVersion)
	}
	if v > LatestSchemaVersion {
		return fmt.Errorf("data schema is at version %d, but this binary only supports up to version %d; update velocirepo", v, LatestSchemaVersion)
	}
	return nil
}

type migration struct {
	description string
	run         func(dataDir string) error
}

var migrations = []migration{
	{
		description: "prefix metric names with daily_ or total_",
		run:         migrate0to1,
	},
	{
		description: "pluralize metric names",
		run:         migrate1to2,
	},
	{
		description: "rename site-level plausible metrics to daily_site_*",
		run:         migrate2to3,
	},
}

func Migrate(dataDir string) (int, error) {
	current, err := SchemaVersion(dataDir)
	if err != nil {
		return 0, err
	}
	return MigrateFrom(dataDir, current)
}

func MigrateFrom(dataDir string, fromVersion int) (int, error) {
	if fromVersion >= LatestSchemaVersion {
		return 0, nil
	}

	applied := 0
	for i := fromVersion; i < LatestSchemaVersion; i++ {
		m := migrations[i]
		if err := m.run(dataDir); err != nil {
			return applied, fmt.Errorf("migration %d (%s): %w", i+1, m.description, err)
		}
		if err := writeSchemaVersion(dataDir, i+1); err != nil {
			return applied, fmt.Errorf("write schema version after migration %d: %w", i+1, err)
		}
		applied++
	}
	return applied, nil
}

func MigrationDescription(version int) string {
	if version < 1 || version > len(migrations) {
		return ""
	}
	return migrations[version-1].description
}

var metricPrefixRenames = map[string]string{
	"downloads":    "daily_downloads",
	"views":        "daily_views",
	"unique_views": "daily_unique_views",
	"clones":       "daily_clones",
	"unique_clones": "daily_unique_clones",
	"pageviews":    "daily_pageviews",
	"visitors":     "daily_visitors",
	"visits":       "daily_visits",
	"reviews":      "total_reviews",
	"rating":       "total_rating",
	"subscribers":  "total_subscribers",
	"channel_views": "total_channel_views",
	"video_count":  "total_video_count",
	"likes":        "total_likes",
	"comments":     "total_comments",
}

var metricPluralRenames = map[string]string{
	"total_rating":      "total_ratings",
	"total_video_count": "total_videos",
}

var metricSiteRenames = map[string]string{
	"daily_pageviews": "daily_site_pageviews",
	"daily_visitors":  "daily_site_visitors",
	"daily_visits":    "daily_site_visits",
}

func migrate0to1(dataDir string) error {
	sourceDirs := []string{"pypi", "cran", "github-traffic", "plausible", "openvsx", "youtube"}
	for _, src := range sourceDirs {
		dir := filepath.Join(dataDir, src)
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			continue
		}
		if err := renameMetricsInDir(dir, metricPrefixRenames); err != nil {
			return fmt.Errorf("%s: %w", src, err)
		}
	}
	return nil
}

func migrate1to2(dataDir string) error {
	sourceDirs := []string{"openvsx", "youtube"}
	for _, src := range sourceDirs {
		dir := filepath.Join(dataDir, src)
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			continue
		}
		if err := renameMetricsInDir(dir, metricPluralRenames); err != nil {
			return fmt.Errorf("%s: %w", src, err)
		}
	}
	return nil
}

func migrate2to3(dataDir string) error {
	dir := filepath.Join(dataDir, "plausible")
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil
	}
	return renameMetricsInDir(dir, metricSiteRenames)
}

func renameMetricsInDir(dir string, renames map[string]string) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(info.Name(), ".jsonl") {
			return nil
		}
		return renameMetricsInFile(path, renames)
	})
}

func renameMetricsInFile(path string, renames map[string]string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	var lines [][]byte
	modified := false
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Bytes()
		var record map[string]interface{}
		if err := json.Unmarshal(line, &record); err != nil {
			lines = append(lines, append([]byte(nil), line...))
			continue
		}

		metric, ok := record["metric"].(string)
		if ok {
			if newName, found := renames[metric]; found {
				record["metric"] = newName
				newLine, err := json.Marshal(record)
				if err != nil {
					lines = append(lines, append([]byte(nil), line...))
					continue
				}
				lines = append(lines, newLine)
				modified = true
				continue
			}
		}
		lines = append(lines, append([]byte(nil), line...))
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	f.Close()

	if !modified {
		return nil
	}

	tmp, err := os.CreateTemp(filepath.Dir(path), ".migrate-*.jsonl")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()

	w := bufio.NewWriter(tmp)
	for _, line := range lines {
		w.Write(line)
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
	return os.Rename(tmpPath, path)
}
