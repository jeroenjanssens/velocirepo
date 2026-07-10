package store

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jeroenjanssens/velocirepo/internal/source"
)

func isTotalMetric(metric string) bool {
	return strings.HasPrefix(metric, "total_")
}

func totalKey(r source.Record) string {
	var b strings.Builder
	b.WriteString(r.Metric)
	b.WriteByte('|')
	b.WriteString(r.Target)
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

func lastRecordedTotals(dir string) (map[string]int64, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var files []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".jsonl") {
			continue
		}
		files = append(files, e.Name())
	}
	if len(files) == 0 {
		return nil, nil
	}

	sort.Strings(files)

	for i := len(files) - 1; i >= 0; i-- {
		path := filepath.Join(dir, files[i])
		records, err := ReadRecords(path)
		if err != nil {
			continue
		}
		totals := make(map[string]int64)
		for _, r := range records {
			if isTotalMetric(r.Metric) {
				totals[totalKey(r)] = r.Value
			}
		}
		if len(totals) > 0 {
			return totals, nil
		}
	}

	return nil, nil
}

func filterUnchangedTotals(records []source.Record, lastValues map[string]int64) []source.Record {
	if len(lastValues) == 0 {
		return records
	}

	result := make([]source.Record, 0, len(records))
	for _, r := range records {
		if !isTotalMetric(r.Metric) {
			result = append(result, r)
			continue
		}
		key := totalKey(r)
		prev, exists := lastValues[key]
		if !exists || r.Value != prev {
			result = append(result, r)
		}
	}
	return result
}
