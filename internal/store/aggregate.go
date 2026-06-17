package store

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/jeroenjanssens/velocirepo/internal/source"
)

func Aggregate(dataDir string, now time.Time) error {
	sourceDirs, err := os.ReadDir(dataDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read data dir: %w", err)
	}

	for _, sourceEntry := range sourceDirs {
		if !sourceEntry.IsDir() {
			continue
		}
		sourcePath := filepath.Join(dataDir, sourceEntry.Name())

		projectDirs, err := os.ReadDir(sourcePath)
		if err != nil {
			continue
		}

		for _, projEntry := range projectDirs {
			if !projEntry.IsDir() {
				continue
			}
			projPath := filepath.Join(sourcePath, projEntry.Name())
			var err error
			if sourceEntry.Name() == "github-events" {
				err = aggregateGitHubEventsProject(projPath, now)
			} else {
				err = aggregateProject(projPath, now)
			}
			if err != nil {
				slog.Warn("aggregate failed", "path", projPath, "error", err)
			}
		}
	}
	return nil
}

func aggregateProject(projDir string, now time.Time) error {
	if err := aggregateDailyToMonthly(projDir, now); err != nil {
		return err
	}
	return aggregateMonthlyToYearly(projDir, now)
}

func aggregateDailyToMonthly(projDir string, now time.Time) error {
	entries, err := os.ReadDir(projDir)
	if err != nil {
		return err
	}

	// Group daily files by year-month
	monthFiles := make(map[string][]string)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if m := dailyPattern.FindStringSubmatch(entry.Name()); m != nil {
			yearMonth := m[1][:7] // "2025-06-01" -> "2025-06"
			monthFiles[yearMonth] = append(monthFiles[yearMonth], filepath.Join(projDir, entry.Name()))
		}
	}

	for yearMonth, files := range monthFiles {
		if !isMonthComplete(yearMonth, now) {
			continue
		}

		sort.Strings(files)

		records, err := readAllFiles(files)
		if err != nil {
			return fmt.Errorf("read files for %s: %w", yearMonth, err)
		}

		records = dedup(records)
		sort.Slice(records, func(i, j int) bool {
			return records[i].Date < records[j].Date
		})

		outPath := filepath.Join(projDir, yearMonth+".jsonl")
		if err := writeFileAtomic(outPath, records); err != nil {
			return err
		}

		for _, f := range files {
			if err := os.Remove(f); err != nil {
				slog.Warn("remove source file", "path", f, "error", err)
			}
		}

		slog.Debug("aggregated daily to monthly", "month", yearMonth, "files", len(files))
	}

	return nil
}

func aggregateMonthlyToYearly(projDir string, now time.Time) error {
	entries, err := os.ReadDir(projDir)
	if err != nil {
		return err
	}

	yearFiles := make(map[string][]string)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if m := monthlyPattern.FindStringSubmatch(entry.Name()); m != nil {
			year := m[1][:4]
			yearFiles[year] = append(yearFiles[year], filepath.Join(projDir, entry.Name()))
		}
	}

	for year, files := range yearFiles {
		if !isYearComplete(year, now) {
			continue
		}

		sort.Strings(files)

		records, err := readAllFiles(files)
		if err != nil {
			return fmt.Errorf("read files for %s: %w", year, err)
		}

		records = dedup(records)
		sort.Slice(records, func(i, j int) bool {
			return records[i].Date < records[j].Date
		})

		outPath := filepath.Join(projDir, year+".jsonl")
		if err := writeFileAtomic(outPath, records); err != nil {
			return err
		}

		for _, f := range files {
			if err := os.Remove(f); err != nil {
				slog.Warn("remove source file", "path", f, "error", err)
			}
		}

		slog.Debug("aggregated monthly to yearly", "year", year, "files", len(files))
	}

	return nil
}

func isMonthComplete(yearMonth string, now time.Time) bool {
	t, err := time.Parse("2006-01", yearMonth)
	if err != nil {
		return false
	}
	endOfMonth := t.AddDate(0, 1, 0)
	return now.After(endOfMonth)
}

func isYearComplete(year string, now time.Time) bool {
	t, err := time.Parse("2006", year)
	if err != nil {
		return false
	}
	endOfYear := t.AddDate(1, 0, 0)
	return now.After(endOfYear)
}

func readAllFiles(paths []string) ([]source.Record, error) {
	var all []source.Record
	for _, p := range paths {
		records, err := ReadRecords(p)
		if err != nil {
			return nil, err
		}
		all = append(all, records...)
	}
	return all, nil
}

func dedup(records []source.Record) []source.Record {
	seen := make(map[string]struct{})
	var result []source.Record

	for _, r := range records {
		key := dedupKey(r)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, r)
	}
	return result
}

func dedupKey(r source.Record) string {
	var b strings.Builder
	b.WriteString(r.Source)
	b.WriteByte('|')
	b.WriteString(r.ProjectID)
	b.WriteByte('|')
	b.WriteString(r.Metric)
	b.WriteByte('|')
	b.WriteString(r.Date)

	if len(r.Tags) > 0 {
		keys := make([]string, 0, len(r.Tags))
		for k := range r.Tags {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			b.WriteByte('|')
			b.WriteString(k)
			b.WriteByte('=')
			b.WriteString(r.Tags[k])
		}
	}

	return b.String()
}

func aggregateGitHubEventsProject(projDir string, now time.Time) error {
	if err := aggregateEventsDailyToMonthly(projDir, now); err != nil {
		return err
	}
	return aggregateEventsMonthlyToYearly(projDir, now)
}

func aggregateEventsDailyToMonthly(projDir string, now time.Time) error {
	entries, err := os.ReadDir(projDir)
	if err != nil {
		return err
	}

	monthFiles := make(map[string][]string)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if m := dailyPattern.FindStringSubmatch(entry.Name()); m != nil {
			yearMonth := m[1][:7]
			monthFiles[yearMonth] = append(monthFiles[yearMonth], filepath.Join(projDir, entry.Name()))
		}
	}

	for yearMonth, files := range monthFiles {
		if !isMonthComplete(yearMonth, now) {
			continue
		}

		sort.Strings(files)

		events, err := readAllEventFiles(files)
		if err != nil {
			return fmt.Errorf("read event files for %s: %w", yearMonth, err)
		}

		events = dedupEvents(events)
		sort.Slice(events, func(i, j int) bool {
			return events[i].Datetime < events[j].Datetime
		})

		outPath := filepath.Join(projDir, yearMonth+".jsonl")
		if err := writeEventsFileAtomic(outPath, events); err != nil {
			return err
		}

		for _, f := range files {
			if err := os.Remove(f); err != nil {
				slog.Warn("remove source file", "path", f, "error", err)
			}
		}

		slog.Debug("aggregated events daily to monthly", "month", yearMonth, "files", len(files))
	}

	return nil
}

func aggregateEventsMonthlyToYearly(projDir string, now time.Time) error {
	entries, err := os.ReadDir(projDir)
	if err != nil {
		return err
	}

	yearFiles := make(map[string][]string)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if m := monthlyPattern.FindStringSubmatch(entry.Name()); m != nil {
			year := m[1][:4]
			yearFiles[year] = append(yearFiles[year], filepath.Join(projDir, entry.Name()))
		}
	}

	for year, files := range yearFiles {
		if !isYearComplete(year, now) {
			continue
		}

		sort.Strings(files)

		events, err := readAllEventFiles(files)
		if err != nil {
			return fmt.Errorf("read event files for %s: %w", year, err)
		}

		events = dedupEvents(events)
		sort.Slice(events, func(i, j int) bool {
			return events[i].Datetime < events[j].Datetime
		})

		outPath := filepath.Join(projDir, year+".jsonl")
		if err := writeEventsFileAtomic(outPath, events); err != nil {
			return err
		}

		for _, f := range files {
			if err := os.Remove(f); err != nil {
				slog.Warn("remove source file", "path", f, "error", err)
			}
		}

		slog.Debug("aggregated events monthly to yearly", "year", year, "files", len(files))
	}

	return nil
}

func readAllEventFiles(paths []string) ([]source.GitHubEvent, error) {
	var all []source.GitHubEvent
	for _, p := range paths {
		events, err := ReadGitHubEvents(p)
		if err != nil {
			return nil, err
		}
		all = append(all, events...)
	}
	return all, nil
}

func dedupEvents(events []source.GitHubEvent) []source.GitHubEvent {
	seen := make(map[string]struct{})
	var result []source.GitHubEvent

	for _, e := range events {
		key := dedupEventKey(e)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, e)
	}
	return result
}

func dedupEventKey(e source.GitHubEvent) string {
	var b strings.Builder
	b.WriteString(e.Source)
	b.WriteByte('|')
	b.WriteString(e.ProjectID)
	b.WriteByte('|')
	b.WriteString(e.EventType)
	b.WriteByte('|')
	b.WriteString(e.Datetime)
	b.WriteByte('|')
	b.WriteString(e.User)
	return b.String()
}

