package store

import "path/filepath"

const (
	MetricsDir     = "metrics"
	EventsDir      = "events"
	ContentDataDir = "content"
	WatermarksDir  = "watermarks"
)

func MetricsProjectDir(dataDir, sourceName, projectID string) string {
	return filepath.Join(dataDir, MetricsDir, sourceName, projectID)
}

func EventsProjectDir(dataDir, sourceName, projectID string) string {
	return filepath.Join(dataDir, EventsDir, sourceName, projectID)
}

func ContentProjectDir(dataDir, sourceName, projectID string) string {
	return filepath.Join(dataDir, ContentDataDir, sourceName, projectID)
}

func SourceProjectDir(dataDir, sourceName, projectID string) string {
	return filepath.Join(dataDir, SourceCategory(sourceName), sourceName, projectID)
}

func WatermarkProjectDir(dataDir, category, sourceName, projectID string) string {
	return filepath.Join(dataDir, WatermarksDir, category, sourceName, projectID)
}
