package fetch

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/posit-dev/velocirepo/internal/config"
	"github.com/posit-dev/velocirepo/internal/source"
	"github.com/posit-dev/velocirepo/internal/store"
)

type fakeMetricSource struct {
	name   string
	target string
	err    error
}

func (s fakeMetricSource) Name() string {
	return s.name
}

func (s fakeMetricSource) Fetch(ctx context.Context, opts source.FetchOptions) ([]source.Record, error) {
	if s.err != nil {
		return nil, s.err
	}
	return []source.Record{{
		Metric:    "daily_downloads",
		ProjectID: opts.ProjectID,
		Target:    s.target,
		Date:      opts.StartDate.Format("2006-01-02"),
		Value:     1,
	}}, nil
}

func TestSourceDoesNotAdvanceCheckpointOnPartialTargetFailure(t *testing.T) {
	dataDir := t.TempDir()
	cfg := &config.Config{
		Data: config.DataConfig{Dir: dataDir},
		Projects: map[string]config.Project{
			"proj": {PyPI: config.StringList{"pkg-a", "pkg-b"}},
		},
	}

	results, err := Source(context.Background(), cfg, Tokens{}, "pypi", Options{
		StartDate:     "2026-06-01",
		EndDate:       "2026-06-01",
		NoConcatenate: true,
	}, func(id string, p config.Project) []source.Source {
		return []source.Source{
			fakeMetricSource{name: "pypi", target: "pkg-a"},
			fakeMetricSource{name: "pypi", target: "pkg-b", err: errors.New("boom")},
		}
	})
	if err != nil {
		t.Fatalf("Source failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1: %#v", len(results), results)
	}
	if results[0].Error != "boom" {
		t.Fatalf("Error = %q, want boom", results[0].Error)
	}

	path := filepath.Join(dataDir, "metrics", "pypi", "proj", "2026-06-01.jsonl")
	if _, err := store.ReadRecords(path); err == nil {
		t.Fatalf("records were written despite partial target failure")
	}
}

func TestSourceWritesAllTargetsForProjectSource(t *testing.T) {
	dataDir := t.TempDir()
	cfg := &config.Config{
		Data: config.DataConfig{Dir: dataDir},
		Projects: map[string]config.Project{
			"proj": {PyPI: config.StringList{"pkg-a", "pkg-b"}},
		},
	}

	results, err := Source(context.Background(), cfg, Tokens{}, "pypi", Options{
		StartDate:     "2026-06-01",
		EndDate:       "2026-06-01",
		NoConcatenate: true,
	}, func(id string, p config.Project) []source.Source {
		var sources []source.Source
		for _, pkg := range p.PyPI {
			sources = append(sources, fakeMetricSource{name: "pypi", target: pkg})
		}
		return sources
	})
	if err != nil {
		t.Fatalf("Source failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1: %#v", len(results), results)
	}
	if results[0].Error != "" || results[0].Skipped != "" {
		t.Fatalf("unexpected result: %#v", results[0])
	}
	if results[0].Records != 2 {
		t.Fatalf("Records = %d, want 2", results[0].Records)
	}

	path := filepath.Join(dataDir, "metrics", "pypi", "proj", "2026-06-01.jsonl")
	records, err := store.ReadRecords(path)
	if err != nil {
		t.Fatalf("ReadRecords failed: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("got %d records, want 2: %#v", len(records), records)
	}

	gotTargets := map[string]bool{}
	for _, record := range records {
		gotTargets[record.Target] = true
	}
	for _, want := range []string{"pkg-a", "pkg-b"} {
		if !gotTargets[want] {
			t.Fatalf("missing target %q in records: %#v", want, records)
		}
	}
}
