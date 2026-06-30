package store

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/jeroenjanssens/velocirepo/internal/dateutil"
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
		dir := MetricsProjectDir(dataDir, sourceName, projectID)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("create directory %s: %w", dir, err)
		}

		path := filepath.Join(dir, date+".jsonl")
		if err := writeFileAtomic(path, dateRecords); err != nil {
			return err
		}
	}

	ensureSchemaVersion(dataDir)
	return nil
}

func writeFileAtomic(path string, records []source.Record) error {
	return writeJSONLAtomic(path, records, "record")
}

func ReadRecords(path string) ([]source.Record, error) {
	return readJSONL[source.Record](path, readJSONLOptions{wrapErrors: true})
}

func LastDate(dataDir, sourceName, projectID string) (time.Time, error) {
	dir := SourceProjectDir(dataDir, sourceName, projectID)
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
			dates = append(dates, dateutil.LastDayOfMonth(m[1]))
		} else if m := yearlyPattern.FindStringSubmatch(name); m != nil {
			dates = append(dates, m[1]+"-12-31")
		}
	}

	if len(dates) == 0 {
		return time.Time{}, nil
	}

	sort.Strings(dates)
	latest := dates[len(dates)-1]

	t, err := dateutil.ParseDate(latest)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse date %s: %w", latest, err)
	}
	return t, nil
}

func groupByDate(records []source.Record) map[string][]source.Record {
	return groupBy(records, func(r source.Record) string { return r.Date })
}

func WriteEvents(dataDir, sourceName, projectID string, events []source.Event) error {
	for i := range events {
		events[i].Source = sourceName
	}

	grouped := groupEventsByDate(events)

	for date, dateEvents := range grouped {
		dir := EventsProjectDir(dataDir, sourceName, projectID)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("create directory %s: %w", dir, err)
		}

		path := filepath.Join(dir, date+".jsonl")
		if err := writeEventsFileAtomic(path, dateEvents); err != nil {
			return err
		}
	}

	ensureSchemaVersion(dataDir)
	return nil
}

func writeEventsFileAtomic(path string, events []source.Event) error {
	return writeJSONLAtomic(path, events, "event")
}

func ReadEvents(path string) ([]source.Event, error) {
	return readJSONL[source.Event](path, readJSONLOptions{wrapErrors: true})
}

func groupEventsByDate(events []source.Event) map[string][]source.Event {
	return groupBy(events, func(e source.Event) string {
		date := e.Datetime[:10]
		return date
	})
}

type DirStats struct {
	LastDate string
	Records  int
	Size     int64
}

func ScanProjectDir(dir string) DirStats {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return DirStats{}
	}

	var stats DirStats
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".jsonl") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		recs, _ := ReadRecords(path)
		stats.Records += len(recs)

		if info, err := e.Info(); err == nil {
			stats.Size += info.Size()
		}

		datePart := strings.TrimSuffix(e.Name(), ".jsonl")
		if datePart > stats.LastDate {
			stats.LastDate = datePart
		}
	}
	return stats
}
