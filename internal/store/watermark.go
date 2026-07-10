package store

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/jeroenjanssens/velocirepo/internal/source"
)

type metricWatermark struct {
	Source    string            `json:"source"`
	ProjectID string            `json:"project_id"`
	Target    string            `json:"target"`
	Metric    string            `json:"metric"`
	Date      string            `json:"date"`
	Tags      map[string]string `json:"tags,omitempty"`
}

func writeMetricWatermarks(dataDir, sourceName, projectID, date string, records []source.Record) error {
	watermarks := metricWatermarks(sourceName, projectID, date, records)
	if len(watermarks) == 0 {
		return nil
	}

	dir := WatermarkProjectDir(dataDir, MetricsDir, sourceName, projectID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create directory %s: %w", dir, err)
	}

	path := filepath.Join(dir, date+".jsonl")
	if err := writeJSONLAtomic(path, watermarks, "metric watermark"); err != nil {
		return err
	}
	return nil
}

func metricWatermarks(sourceName, projectID, date string, records []source.Record) []metricWatermark {
	seen := make(map[string]metricWatermark)
	for _, r := range records {
		if !isTotalMetric(r.Metric) {
			continue
		}
		key := totalKey(r)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = metricWatermark{
			Source:    sourceName,
			ProjectID: projectID,
			Target:    r.Target,
			Metric:    r.Metric,
			Date:      date,
			Tags:      cloneStringMap(r.Tags),
		}
	}
	if len(seen) == 0 {
		return nil
	}

	keys := make([]string, 0, len(seen))
	for key := range seen {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	watermarks := make([]metricWatermark, 0, len(keys))
	for _, key := range keys {
		watermarks = append(watermarks, seen[key])
	}
	return watermarks
}

func cloneStringMap(m map[string]string) map[string]string {
	if len(m) == 0 {
		return nil
	}
	clone := make(map[string]string, len(m))
	for k, v := range m {
		clone[k] = v
	}
	return clone
}
