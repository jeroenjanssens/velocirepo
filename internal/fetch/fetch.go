package fetch

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/jeroenjanssens/velocirepo/internal/config"
	"github.com/jeroenjanssens/velocirepo/internal/source"
	"github.com/jeroenjanssens/velocirepo/internal/store"
	"golang.org/x/sync/errgroup"
)

type Options struct {
	Project       string
	StartDate     string
	EndDate       string
	NoConcatenate bool
}

type Result struct {
	Source    string        `json:"source"`
	ProjectID string       `json:"project_id"`
	Records   int          `json:"records"`
	StartDate string       `json:"start_date"`
	EndDate   string       `json:"end_date"`
	Duration  time.Duration `json:"duration,omitempty"`
	Skipped   string       `json:"skipped,omitempty"`
	Error     string       `json:"error,omitempty"`
}

type Tokens struct {
	GitHub    string
	Plausible string
	YouTube   string
}

func TokensFromEnv() Tokens {
	return Tokens{
		GitHub:    os.Getenv("GITHUB_TOKEN"),
		Plausible: os.Getenv("PLAUSIBLE_TOKEN"),
		YouTube:   os.Getenv("YOUTUBE_TOKEN"),
	}
}

func resolveEndDate(cfg *config.Config, endDateStr string) (time.Time, error) {
	if endDateStr != "" {
		return time.Parse("2006-01-02", endDateStr)
	}
	if cfg.Settings.EndDate == "yesterday" || cfg.Settings.EndDate == "" {
		return time.Now().UTC().AddDate(0, 0, -1).Truncate(24 * time.Hour), nil
	}
	return time.Parse("2006-01-02", cfg.Settings.EndDate)
}

func resolveStartDate(dataDir, sourceName, projectID, startDateStr string) (time.Time, error) {
	if startDateStr != "" {
		return time.Parse("2006-01-02", startDateStr)
	}
	last, err := store.LastDate(dataDir, sourceName, projectID)
	if err != nil {
		return time.Time{}, err
	}
	var start time.Time
	if last.IsZero() {
		start = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	} else {
		start = last.AddDate(0, 0, 1)
	}

	if sourceName == "github-traffic" {
		earliest := time.Now().UTC().AddDate(0, 0, -14).Truncate(24 * time.Hour)
		if start.Before(earliest) {
			start = earliest
		}
	}

	return start, nil
}

func filterProjects(projects map[string]config.Project, projectID string) map[string]config.Project {
	if projectID == "" {
		return projects
	}
	p, ok := projects[projectID]
	if !ok {
		return nil
	}
	return map[string]config.Project{projectID: p}
}

func All(ctx context.Context, cfg *config.Config, tokens Tokens, opts Options) ([]Result, error) {
	projects := cfg.ResolveProjects()
	if projects == nil {
		return nil, fmt.Errorf("no projects configured")
	}

	projects = filterProjects(projects, opts.Project)
	if projects == nil {
		return nil, fmt.Errorf("project %q not found in config", opts.Project)
	}

	endDate, err := resolveEndDate(cfg, opts.EndDate)
	if err != nil {
		return nil, fmt.Errorf("parse end date: %w", err)
	}

	dataDir := cfg.DataDir()
	client := &http.Client{Timeout: 30 * time.Second}

	type fetchJob struct {
		sourceName string
		projectID  string
		src        source.Source
		eventSrc   source.GitHubEventSource
	}

	var jobs []fetchJob
	for id, proj := range projects {
		for _, repo := range proj.GitHubTraffic {
			jobs = append(jobs, fetchJob{sourceName: "github-traffic", projectID: id, src: &source.GitHubTraffic{Client: client, Token: tokens.GitHub, Repo: repo}})
		}
		for _, repo := range proj.GitHubEvents {
			jobs = append(jobs, fetchJob{sourceName: "github", projectID: id, eventSrc: &source.GitHubEvents{Client: client, Token: tokens.GitHub, Repo: repo}})
		}
		for _, pkg := range proj.PyPI {
			jobs = append(jobs, fetchJob{sourceName: "pypi", projectID: id, src: &source.PyPI{Client: client, Package: pkg}})
		}
		for _, pkg := range proj.CRAN {
			jobs = append(jobs, fetchJob{sourceName: "cran", projectID: id, src: &source.CRAN{Client: client, Package: pkg}})
		}
		for _, formula := range proj.Homebrew {
			jobs = append(jobs, fetchJob{sourceName: "homebrew", projectID: id, src: &source.Homebrew{Client: client, Formula: formula}})
		}
		if tokens.Plausible != "" {
			for _, site := range proj.Plausible {
				jobs = append(jobs, fetchJob{sourceName: "plausible", projectID: id, src: &source.Plausible{Client: client, APIKey: tokens.Plausible, SiteID: site}})
			}
		}
		for _, ext := range proj.OpenVSX {
			jobs = append(jobs, fetchJob{sourceName: "openvsx", projectID: id, src: &source.OpenVSX{Client: client, ExtensionID: ext}})
		}
		if tokens.YouTube != "" {
			for _, target := range proj.YouTube {
				jobs = append(jobs, fetchJob{sourceName: "youtube", projectID: id, src: &source.YouTube{Client: client, APIKey: tokens.YouTube, Target: target}})
			}
		}
	}

	var results []Result
	resultsCh := make(chan Result, len(jobs))

	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(4)

	for _, job := range jobs {
		job := job
		g.Go(func() error {
			started := time.Now()
			startDate, err := resolveStartDate(dataDir, job.sourceName, job.projectID, opts.StartDate)
			if err != nil {
				resultsCh <- Result{
					Source:    job.sourceName,
					ProjectID: job.projectID,
					Duration:  time.Since(started),
					Error:     fmt.Sprintf("resolve start date: %v", err),
				}
				return nil
			}

			if !startDate.Before(endDate.AddDate(0, 0, 1)) {
				resultsCh <- Result{
					Source:    job.sourceName,
					ProjectID: job.projectID,
					Duration:  time.Since(started),
					Skipped:   "already up to date",
				}
				return nil
			}

			if job.eventSrc != nil {
				events, err := job.eventSrc.FetchEvents(gctx, source.FetchOptions{
					ProjectID: job.projectID,
					StartDate: startDate,
					EndDate:   endDate,
				})
				if err != nil {
					resultsCh <- Result{
						Source:    job.sourceName,
						ProjectID: job.projectID,
						Duration:  time.Since(started),
						Error:     err.Error(),
					}
					return nil
				}

				if len(events) == 0 {
					resultsCh <- Result{
						Source:    job.sourceName,
						ProjectID: job.projectID,
						Duration:  time.Since(started),
						Skipped:   "no new events",
					}
					return nil
				}

				if err := store.WriteGitHubEvents(dataDir, job.sourceName, job.projectID, events); err != nil {
					resultsCh <- Result{
						Source:    job.sourceName,
						ProjectID: job.projectID,
						Duration:  time.Since(started),
						Error:     fmt.Sprintf("write: %v", err),
					}
					return nil
				}

				resultsCh <- Result{
					Source:    job.sourceName,
					ProjectID: job.projectID,
					Records:   len(events),
					StartDate: startDate.Format("2006-01-02"),
					EndDate:   endDate.Format("2006-01-02"),
					Duration:  time.Since(started),
				}
			} else {
				records, err := job.src.Fetch(gctx, source.FetchOptions{
					ProjectID: job.projectID,
					StartDate: startDate,
					EndDate:   endDate,
				})
				if err != nil {
					resultsCh <- Result{
						Source:    job.sourceName,
						ProjectID: job.projectID,
						Duration:  time.Since(started),
						Error:     err.Error(),
					}
					return nil
				}

				if len(records) == 0 {
					resultsCh <- Result{
						Source:    job.sourceName,
						ProjectID: job.projectID,
						Duration:  time.Since(started),
						Skipped:   "no new records",
					}
					return nil
				}

				if err := store.WriteRecords(dataDir, job.sourceName, job.projectID, records); err != nil {
					resultsCh <- Result{
						Source:    job.sourceName,
						ProjectID: job.projectID,
						Duration:  time.Since(started),
						Error:     fmt.Sprintf("write: %v", err),
					}
					return nil
				}

				if yt, ok := job.src.(*source.YouTube); ok {
					if entries := yt.IndexEntries(); len(entries) > 0 {
						store.WriteYouTubeIndex(dataDir, job.projectID, entries)
					}
				}

				resultsCh <- Result{
					Source:    job.sourceName,
					ProjectID: job.projectID,
					Records:   len(records),
					StartDate: startDate.Format("2006-01-02"),
					EndDate:   endDate.Format("2006-01-02"),
					Duration:  time.Since(started),
				}
			}

			return nil
		})
	}

	g.Wait()
	close(resultsCh)

	for r := range resultsCh {
		results = append(results, r)
	}

	if !opts.NoConcatenate {
		if err := store.Aggregate(dataDir, time.Now().UTC()); err != nil {
			return results, fmt.Errorf("concatenation: %w", err)
		}
	}

	return results, nil
}

func Source(ctx context.Context, cfg *config.Config, tokens Tokens, sourceName string, opts Options, createSources func(id string, proj config.Project) []source.Source) ([]Result, error) {
	projects := cfg.ResolveProjects()
	if projects == nil {
		return nil, fmt.Errorf("no projects configured")
	}

	projects = filterProjects(projects, opts.Project)
	if projects == nil {
		return nil, fmt.Errorf("project %q not found in config", opts.Project)
	}

	endDate, err := resolveEndDate(cfg, opts.EndDate)
	if err != nil {
		return nil, fmt.Errorf("parse end date: %w", err)
	}

	dataDir := cfg.DataDir()
	var results []Result

	for id, proj := range projects {
		sources := createSources(id, proj)
		if len(sources) == 0 {
			continue
		}

		startDate, err := resolveStartDate(dataDir, sourceName, id, opts.StartDate)
		if err != nil {
			results = append(results, Result{
				Source:    sourceName,
				ProjectID: id,
				Error:     fmt.Sprintf("resolve start date: %v", err),
			})
			continue
		}

		if !startDate.Before(endDate.AddDate(0, 0, 1)) {
			results = append(results, Result{
				Source:    sourceName,
				ProjectID: id,
				Skipped:   "already up to date",
			})
			continue
		}

		for _, src := range sources {
			started := time.Now()
			records, err := src.Fetch(ctx, source.FetchOptions{
				ProjectID: id,
				StartDate: startDate,
				EndDate:   endDate,
			})
			if err != nil {
				results = append(results, Result{
					Source:    sourceName,
					ProjectID: id,
					Duration:  time.Since(started),
					Error:     err.Error(),
				})
				continue
			}

			if len(records) == 0 {
				results = append(results, Result{
					Source:    sourceName,
					ProjectID: id,
					Duration:  time.Since(started),
					Skipped:   "no new records",
				})
				continue
			}

			if err := store.WriteRecords(dataDir, sourceName, id, records); err != nil {
				results = append(results, Result{
					Source:    sourceName,
					ProjectID: id,
					Duration:  time.Since(started),
					Error:     fmt.Sprintf("write: %v", err),
				})
				continue
			}

			if yt, ok := src.(*source.YouTube); ok {
				if entries := yt.IndexEntries(); len(entries) > 0 {
					store.WriteYouTubeIndex(dataDir, id, entries)
				}
			}

			results = append(results, Result{
				Source:    sourceName,
				ProjectID: id,
				Records:   len(records),
				StartDate: startDate.Format("2006-01-02"),
				EndDate:   endDate.Format("2006-01-02"),
				Duration:  time.Since(started),
			})
		}
	}

	if !opts.NoConcatenate {
		if err := store.Aggregate(dataDir, time.Now().UTC()); err != nil {
			return results, fmt.Errorf("concatenation: %w", err)
		}
	}

	return results, nil
}

func GitHub(ctx context.Context, cfg *config.Config, tokens Tokens, opts Options) ([]Result, error) {
	projects := cfg.ResolveProjects()
	if projects == nil {
		return nil, fmt.Errorf("no projects configured")
	}

	projects = filterProjects(projects, opts.Project)
	if projects == nil {
		return nil, fmt.Errorf("project %q not found in config", opts.Project)
	}

	endDate, err := resolveEndDate(cfg, opts.EndDate)
	if err != nil {
		return nil, fmt.Errorf("parse end date: %w", err)
	}

	dataDir := cfg.DataDir()
	client := &http.Client{Timeout: 30 * time.Second}
	var results []Result

	for id, proj := range projects {
		for _, repo := range proj.GitHubEvents {
			started := time.Now()
			startDate, err := resolveStartDate(dataDir, "github", id, opts.StartDate)
			if err != nil {
				results = append(results, Result{
					Source:    "github",
					ProjectID: id,
					Duration:  time.Since(started),
					Error:     fmt.Sprintf("resolve start date: %v", err),
				})
				continue
			}

			if !startDate.Before(endDate.AddDate(0, 0, 1)) {
				results = append(results, Result{
					Source:    "github",
					ProjectID: id,
					Duration:  time.Since(started),
					Skipped:   "already up to date",
				})
				continue
			}

			src := &source.GitHubEvents{Client: client, Token: tokens.GitHub, Repo: repo}
			events, err := src.FetchEvents(ctx, source.FetchOptions{
				ProjectID: id,
				StartDate: startDate,
				EndDate:   endDate,
			})
			if err != nil {
				results = append(results, Result{
					Source:    "github",
					ProjectID: id,
					Duration:  time.Since(started),
					Error:     err.Error(),
				})
				continue
			}

			if len(events) == 0 {
				results = append(results, Result{
					Source:    "github",
					ProjectID: id,
					Duration:  time.Since(started),
					Skipped:   "no new events",
				})
				continue
			}

			if err := store.WriteGitHubEvents(dataDir, "github", id, events); err != nil {
				results = append(results, Result{
					Source:    "github",
					ProjectID: id,
					Duration:  time.Since(started),
					Error:     fmt.Sprintf("write: %v", err),
				})
				continue
			}

			results = append(results, Result{
				Source:    "github",
				ProjectID: id,
				Records:   len(events),
				StartDate: startDate.Format("2006-01-02"),
				EndDate:   endDate.Format("2006-01-02"),
				Duration:  time.Since(started),
			})
		}
	}

	if !opts.NoConcatenate {
		if err := store.Aggregate(dataDir, time.Now().UTC()); err != nil {
			return results, fmt.Errorf("concatenation: %w", err)
		}
	}

	return results, nil
}

func Traffic(ctx context.Context, cfg *config.Config, tokens Tokens, opts Options) ([]Result, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	return Source(ctx, cfg, tokens, "github-traffic", opts, func(id string, p config.Project) []source.Source {
		var sources []source.Source
		for _, repo := range p.GitHubTraffic {
			sources = append(sources, &source.GitHubTraffic{Client: client, Token: tokens.GitHub, Repo: repo})
		}
		return sources
	})
}

func PyPI(ctx context.Context, cfg *config.Config, tokens Tokens, opts Options) ([]Result, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	return Source(ctx, cfg, tokens, "pypi", opts, func(id string, p config.Project) []source.Source {
		var sources []source.Source
		for _, pkg := range p.PyPI {
			sources = append(sources, &source.PyPI{Client: client, Package: pkg})
		}
		return sources
	})
}

func CRAN(ctx context.Context, cfg *config.Config, tokens Tokens, opts Options) ([]Result, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	return Source(ctx, cfg, tokens, "cran", opts, func(id string, p config.Project) []source.Source {
		var sources []source.Source
		for _, pkg := range p.CRAN {
			sources = append(sources, &source.CRAN{Client: client, Package: pkg})
		}
		return sources
	})
}

func Homebrew(ctx context.Context, cfg *config.Config, tokens Tokens, opts Options) ([]Result, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	return Source(ctx, cfg, tokens, "homebrew", opts, func(id string, p config.Project) []source.Source {
		var sources []source.Source
		for _, formula := range p.Homebrew {
			sources = append(sources, &source.Homebrew{Client: client, Formula: formula})
		}
		return sources
	})
}

func Plausible(ctx context.Context, cfg *config.Config, tokens Tokens, opts Options) ([]Result, error) {
	if tokens.Plausible == "" {
		return nil, fmt.Errorf("PLAUSIBLE_TOKEN not set")
	}
	client := &http.Client{Timeout: 30 * time.Second}
	return Source(ctx, cfg, tokens, "plausible", opts, func(id string, p config.Project) []source.Source {
		var sources []source.Source
		for _, site := range p.Plausible {
			sources = append(sources, &source.Plausible{Client: client, APIKey: tokens.Plausible, SiteID: site})
		}
		return sources
	})
}

func OpenVSX(ctx context.Context, cfg *config.Config, tokens Tokens, opts Options) ([]Result, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	return Source(ctx, cfg, tokens, "openvsx", opts, func(id string, p config.Project) []source.Source {
		var sources []source.Source
		for _, ext := range p.OpenVSX {
			sources = append(sources, &source.OpenVSX{Client: client, ExtensionID: ext})
		}
		return sources
	})
}

func YouTube(ctx context.Context, cfg *config.Config, tokens Tokens, opts Options) ([]Result, error) {
	if tokens.YouTube == "" {
		return nil, fmt.Errorf("YOUTUBE_TOKEN not set")
	}
	client := &http.Client{Timeout: 30 * time.Second}
	return Source(ctx, cfg, tokens, "youtube", opts, func(id string, p config.Project) []source.Source {
		var sources []source.Source
		for _, target := range p.YouTube {
			sources = append(sources, &source.YouTube{Client: client, APIKey: tokens.YouTube, Target: target})
		}
		return sources
	})
}
