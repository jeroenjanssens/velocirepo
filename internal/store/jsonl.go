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
		f, err := os.Create(path)
		if err != nil {
			return fmt.Errorf("create file %s: %w", path, err)
		}

		w := bufio.NewWriter(f)
		for _, r := range dateRecords {
			data, err := json.Marshal(r)
			if err != nil {
				f.Close()
				return fmt.Errorf("marshal record: %w", err)
			}
			w.Write(data)
			w.WriteByte('\n')
		}

		if err := w.Flush(); err != nil {
			f.Close()
			return fmt.Errorf("flush %s: %w", path, err)
		}
		if err := f.Close(); err != nil {
			return fmt.Errorf("close %s: %w", path, err)
		}
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
