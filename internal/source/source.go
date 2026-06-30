package source

import (
	"context"
	"encoding/json"
	"math"
	"time"
)

type Record struct {
	Source    string            `json:"source"`
	Metric   string            `json:"metric"`
	ProjectID string           `json:"project_id"`
	Target   string            `json:"target"`
	Date     string            `json:"date"`
	Value    int64             `json:"value"`
	Tags     map[string]string `json:"tags,omitempty"`
}

func (r *Record) UnmarshalJSON(data []byte) error {
	type Alias Record
	aux := &struct {
		Value json.Number `json:"value"`
		*Alias
	}{
		Alias: (*Alias)(r),
	}
	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}
	if aux.Value != "" {
		if v, err := aux.Value.Int64(); err == nil {
			r.Value = v
		} else if f, err := aux.Value.Float64(); err == nil {
			r.Value = int64(math.Round(f))
		}
	}
	return nil
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

type Event struct {
	Source    string            `json:"source"`
	Type      string            `json:"type"`
	ProjectID string            `json:"project_id"`
	Target    string            `json:"target"`
	Datetime  string            `json:"datetime"`
	Tags      map[string]string `json:"tags,omitempty"`
}

type EventSource interface {
	Name() string
	FetchEvents(ctx context.Context, opts FetchOptions) ([]Event, error)
}
