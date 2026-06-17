package store

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"time"

	"github.com/jeroenjanssens/velocirepo/internal/source"
)

var (
	dailyPattern   = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2})\.jsonl$`)
	monthlyPattern = regexp.MustCompile(`^(\d{4}-\d{2})\.jsonl$`)
	yearlyPattern  = regexp.MustCompile(`^(\d{4})\.jsonl$`)
)

func WriteRecords(dataDir, sourceName, projectID string, records []source.Record) error {
	for i := range records {
		records[i].Source = sourceName
	}

	grouped := groupByDate(records)

	for date, dateRecords := range grouped {
		dir := filepath.Join(dataDir, sourceName, projectID)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("create directory %s: %w", dir, err)
		}

		path := filepath.Join(dir, date+".jsonl")
		if err := writeFileAtomic(path, dateRecords); err != nil {
			return err
		}
	}

	return nil
}

func writeFileAtomic(path string, records []source.Record) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".tmp-*.jsonl")
	if err != nil {
		return fmt.Errorf("create temp file in %s: %w", dir, err)
	}
	tmpPath := tmp.Name()

	w := bufio.NewWriter(tmp)
	for _, r := range records {
		data, err := json.Marshal(r)
		if err != nil {
			tmp.Close()
			os.Remove(tmpPath)
			return fmt.Errorf("marshal record: %w", err)
		}
		w.Write(data)
		w.WriteByte('\n')
	}

	if err := w.Flush(); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("flush %s: %w", tmpPath, err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("close %s: %w", tmpPath, err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("rename %s -> %s: %w", tmpPath, path, err)
	}
	return nil
}

func ReadRecords(path string) ([]source.Record, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	var records []source.Record
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		var r source.Record
		if err := json.Unmarshal(scanner.Bytes(), &r); err != nil {
			return nil, fmt.Errorf("unmarshal line in %s: %w", path, err)
		}
		records = append(records, r)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan %s: %w", path, err)
	}
	return records, nil
}

func LastDate(dataDir, sourceName, projectID string) (time.Time, error) {
	dir := filepath.Join(dataDir, sourceName, projectID)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return time.Time{}, nil
		}
		return time.Time{}, fmt.Errorf("read directory %s: %w", dir, err)
	}

	var dates []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()

		if m := dailyPattern.FindStringSubmatch(name); m != nil {
			dates = append(dates, m[1])
		} else if m := monthlyPattern.FindStringSubmatch(name); m != nil {
			dates = append(dates, lastDayOfMonth(m[1]))
		} else if m := yearlyPattern.FindStringSubmatch(name); m != nil {
			dates = append(dates, m[1]+"-12-31")
		}
	}

	if len(dates) == 0 {
		return time.Time{}, nil
	}

	sort.Strings(dates)
	latest := dates[len(dates)-1]

	t, err := time.Parse("2006-01-02", latest)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse date %s: %w", latest, err)
	}
	return t, nil
}

func lastDayOfMonth(yearMonth string) string {
	t, err := time.Parse("2006-01", yearMonth)
	if err != nil {
		return yearMonth + "-28"
	}
	last := t.AddDate(0, 1, -1)
	return last.Format("2006-01-02")
}

func groupByDate(records []source.Record) map[string][]source.Record {
	grouped := make(map[string][]source.Record)
	for _, r := range records {
		grouped[r.Date] = append(grouped[r.Date], r)
	}
	return grouped
}

func WriteGitHubEvents(dataDir, sourceName, projectID string, events []source.GitHubEvent) error {
	for i := range events {
		events[i].Source = sourceName
	}

	grouped := groupEventsByDate(events)

	for date, dateEvents := range grouped {
		dir := filepath.Join(dataDir, sourceName, projectID)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("create directory %s: %w", dir, err)
		}

		path := filepath.Join(dir, date+".jsonl")
		if err := writeEventsFileAtomic(path, dateEvents); err != nil {
			return err
		}
	}

	return nil
}

func writeEventsFileAtomic(path string, events []source.GitHubEvent) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".tmp-*.jsonl")
	if err != nil {
		return fmt.Errorf("create temp file in %s: %w", dir, err)
	}
	tmpPath := tmp.Name()

	w := bufio.NewWriter(tmp)
	for _, e := range events {
		data, err := json.Marshal(e)
		if err != nil {
			tmp.Close()
			os.Remove(tmpPath)
			return fmt.Errorf("marshal event: %w", err)
		}
		w.Write(data)
		w.WriteByte('\n')
	}

	if err := w.Flush(); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return fmt.Errorf("flush %s: %w", tmpPath, err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("close %s: %w", tmpPath, err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("rename %s -> %s: %w", tmpPath, path, err)
	}
	return nil
}

func ReadGitHubEvents(path string) ([]source.GitHubEvent, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	var events []source.GitHubEvent
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		var e source.GitHubEvent
		if err := json.Unmarshal(scanner.Bytes(), &e); err != nil {
			return nil, fmt.Errorf("unmarshal line in %s: %w", path, err)
		}
		events = append(events, e)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan %s: %w", path, err)
	}
	return events, nil
}

func groupEventsByDate(events []source.GitHubEvent) map[string][]source.GitHubEvent {
	grouped := make(map[string][]source.GitHubEvent)
	for _, e := range events {
		date := e.Datetime[:10]
		grouped[date] = append(grouped[date], e)
	}
	return grouped
}
