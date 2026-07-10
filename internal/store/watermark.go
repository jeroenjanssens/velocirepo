package store

import (
	"fmt"
	"os"
	"path/filepath"
)

type metricWatermark struct {
	Source    string `json:"source"`
	ProjectID string `json:"project_id"`
	Date      string `json:"date"`
}

func writeMetricWatermark(dataDir, sourceName, projectID, date string) error {
	dir := WatermarkProjectDir(dataDir, MetricsDir, sourceName, projectID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create directory %s: %w", dir, err)
	}

	path := filepath.Join(dir, date+".jsonl")
	watermarks := []metricWatermark{{
		Source:    sourceName,
		ProjectID: projectID,
		Date:      date,
	}}
	if err := writeJSONLAtomic(path, watermarks, "metric watermark"); err != nil {
		return err
	}
	return nil
}
