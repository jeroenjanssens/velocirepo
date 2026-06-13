package source

import (
	"context"
	"time"
)

type Record struct {
	Metric    string            `json:"metric"`
	ProjectID string            `json:"project_id"`
	Date      string            `json:"date"`
	Value     int64             `json:"value"`
	Tags      map[string]string `json:"tags,omitempty"`
}

type FetchOptions struct {
	ProjectID string
	StartDate time.Time
	EndDate   time.Time
}

type Source interface {
	Name() string
	Fetch(ctx context.Context, opts FetchOptions) ([]Record, error)
}
