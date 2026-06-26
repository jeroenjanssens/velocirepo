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

var EventSources = map[string]bool{
	"github": true,
}

// SourceCategory returns the date-partitioned category ("metrics" or "events")
// for a given source. Sources may also produce content via the ContentProvider
// interface, which is handled separately by the fetch orchestrator.
func SourceCategory(sourceName string) string {
	if EventSources[sourceName] {
		return "events"
	}
	return "metrics"
}

// Aggregate concatenates daily→monthly→yearly for all projects. Errors for
// individual projects are logged but not propagated; category directories that
// don't exist yet are silently skipped.
func Aggregate(dataDir string, now time.Time) error {
	aggregateCategory(filepath.Join(dataDir, "metrics"), now, false)
	aggregateCategory(filepath.Join(dataDir, "events"), now, true)
	return nil
}

func aggregateCategory(categoryDir string, now time.Time, isEvents bool) {
	sourceDirs, err := os.ReadDir(categoryDir)
	if err != nil {
		return
	}

	for _, sourceEntry := range sourceDirs {
		if !sourceEntry.IsDir() {
			continue
		}
		sourcePath := filepath.Join(categoryDir, sourceEntry.Name())

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
			if isEvents {
				err = aggregateEventsProject(projPath, now)
			} else {
				err = aggregateProject(projPath, now)
			}
			if err != nil {
				slog.Warn("concatenation failed", "path", projPath, "error", err)
			}
		}
	}
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

		slog.Debug("concatenated daily to monthly", "month", yearMonth, "files", len(files))
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

		slog.Debug("concatenated monthly to yearly", "year", year, "files", len(files))
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

func aggregateEventsProject(projDir string, now time.Time) error {
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

		slog.Debug("concatenated events daily to monthly", "month", yearMonth, "files", len(files))
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

		slog.Debug("concatenated events monthly to yearly", "year", year, "files", len(files))
	}

	return nil
}

func readAllEventFiles(paths []string) ([]source.Event, error) {
	var all []source.Event
	for _, p := range paths {
		events, err := ReadEvents(p)
		if err != nil {
			return nil, err
		}
		all = append(all, events...)
	}
	return all, nil
}

func dedupEvents(events []source.Event) []source.Event {
	seen := make(map[string]struct{})
	var result []source.Event

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

func dedupEventKey(e source.Event) string {
	var b strings.Builder
	b.WriteString(e.Source)
	b.WriteByte('|')
	b.WriteString(e.ProjectID)
	b.WriteByte('|')
	b.WriteString(e.Type)
	b.WriteByte('|')
	b.WriteString(e.Datetime)

	if len(e.Tags) > 0 {
		keys := make([]string, 0, len(e.Tags))
		for k := range e.Tags {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			b.WriteByte('|')
			b.WriteString(k)
			b.WriteByte('=')
			b.WriteString(e.Tags[k])
		}
	}

	return b.String()
}

