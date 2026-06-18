package source

import (
	"context"
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

type FetchOptions struct {
	ProjectID string
	StartDate time.Time
	EndDate   time.Time
}

type Source interface {
	Name() string
	Fetch(ctx context.Context, opts FetchOptions) ([]Record, error)
}

type GitHubEvent struct {
	Source     string `json:"source"`
	EventType  string `json:"event_type"`
	ProjectID  string `json:"project_id"`
	GitHubRepo string `json:"github_repo"`
	Datetime   string `json:"datetime"`
	User       string `json:"user"`
}

type GitHubEventSource interface {
	Name() string
	FetchEvents(ctx context.Context, opts FetchOptions) ([]GitHubEvent, error)
}
