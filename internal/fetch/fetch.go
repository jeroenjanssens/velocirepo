package fetch

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/jeroenjanssens/velocirepo/internal/auth"
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
	ProjectID string        `json:"project_id"`
	Records   int           `json:"records"`
	StartDate string        `json:"start_date"`
	EndDate   string        `json:"end_date"`
	Duration  time.Duration `json:"duration,omitempty"`
	Skipped   string        `json:"skipped,omitempty"`
	Error     string        `json:"error,omitempty"`
}

type Tokens struct {
	GitHub    string
	Plausible string
	YouTube   string
	LinkedIn  string
}

func TokensFromEnv() Tokens {
	return Tokens{
		GitHub:    os.Getenv("GITHUB_TOKEN"),
		Plausible: os.Getenv("PLAUSIBLE_TOKEN"),
		YouTube:   os.Getenv("YOUTUBE_TOKEN"),
		LinkedIn:  os.Getenv("LINKEDIN_TOKEN"),
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

type fetchJob struct {
	sourceName   string
	projectID    string
	sources      []source.Source
	eventSources []source.EventSource
	startDate    time.Time
	startErr     error
}

type jobKey struct {
	sourceName string
	projectID  string
}

func addMetricJob(jobs map[jobKey]*fetchJob, order *[]jobKey, sourceName, projectID string, src source.Source) {
	key := jobKey{sourceName: sourceName, projectID: projectID}
	job := jobs[key]
	if job == nil {
		job = &fetchJob{sourceName: sourceName, projectID: projectID}
		jobs[key] = job
		*order = append(*order, key)
	}
	job.sources = append(job.sources, src)
}

func addEventJob(jobs map[jobKey]*fetchJob, order *[]jobKey, sourceName, projectID string, src source.EventSource) {
	key := jobKey{sourceName: sourceName, projectID: projectID}
	job := jobs[key]
	if job == nil {
		job = &fetchJob{sourceName: sourceName, projectID: projectID}
		jobs[key] = job
		*order = append(*order, key)
	}
	job.eventSources = append(job.eventSources, src)
}

func orderedJobs(jobs map[jobKey]*fetchJob, order []jobKey) []fetchJob {
	out := make([]fetchJob, 0, len(order))
	for _, key := range order {
		out = append(out, *jobs[key])
	}
	return out
}

func selectedProjects(cfg *config.Config, opts Options) (map[string]config.Project, error) {
	projects := cfg.Projects
	if len(projects) == 0 {
		return nil, fmt.Errorf("no projects configured")
	}

	projects = filterProjects(projects, opts.Project)
	if projects == nil {
		return nil, fmt.Errorf("project %q not found in config", opts.Project)
	}
	return projects, nil
}

func runJobs(ctx context.Context, cfg *config.Config, opts Options, jobs []fetchJob, concurrency int) ([]Result, error) {
	endDate, err := resolveEndDate(cfg, opts.EndDate)
	if err != nil {
		return nil, fmt.Errorf("parse end date: %w", err)
	}

	dataDir := cfg.DataDir()
	for i := range jobs {
		startDate, err := resolveStartDate(dataDir, jobs[i].sourceName, jobs[i].projectID, opts.StartDate)
		jobs[i].startDate = startDate
		jobs[i].startErr = err
	}

	resultsCh := make(chan []Result, len(jobs))

	if concurrency <= 0 {
		concurrency = 1
	}

	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(concurrency)

	for _, job := range jobs {
		job := job
		g.Go(func() error {
			resultsCh <- runJob(gctx, dataDir, endDate, job)
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}
	close(resultsCh)

	results := make([]Result, 0, len(jobs))
	for jobResults := range resultsCh {
		results = append(results, jobResults...)
	}

	if !opts.NoConcatenate {
		if err := store.Aggregate(dataDir, time.Now().UTC()); err != nil {
			return results, fmt.Errorf("concatenation: %w", err)
		}
	}

	return results, nil
}

func runJob(ctx context.Context, dataDir string, endDate time.Time, job fetchJob) []Result {
	started := time.Now()
	if job.startErr != nil {
		return []Result{{
			Source:    job.sourceName,
			ProjectID: job.projectID,
			Duration:  time.Since(started),
			Error:     fmt.Sprintf("resolve start date: %v", job.startErr),
		}}
	}

	if !job.startDate.Before(endDate.AddDate(0, 0, 1)) {
		return []Result{{
			Source:    job.sourceName,
			ProjectID: job.projectID,
			Duration:  time.Since(started),
			Skipped:   "already up to date",
		}}
	}

	if len(job.eventSources) > 0 {
		return runEventJob(ctx, dataDir, job, source.FetchOptions{
			ProjectID: job.projectID,
			StartDate: job.startDate,
			EndDate:   endDate,
		}, started)
	}

	if len(job.sources) == 0 {
		return []Result{{
			Source:    job.sourceName,
			ProjectID: job.projectID,
			Duration:  time.Since(started),
			Error:     "no fetcher configured",
		}}
	}

	return runMetricJob(ctx, dataDir, job, source.FetchOptions{
		ProjectID: job.projectID,
		StartDate: job.startDate,
		EndDate:   endDate,
	}, started)
}

func runEventJob(ctx context.Context, dataDir string, job fetchJob, opts source.FetchOptions, started time.Time) []Result {
	var results []Result
	var events []source.Event
	for _, eventSrc := range job.eventSources {
		fetched, err := eventSrc.FetchEvents(ctx, opts)
		if err != nil {
			results = append(results, Result{
				Source:    job.sourceName,
				ProjectID: job.projectID,
				Duration:  time.Since(started),
				Error:     err.Error(),
			})
			continue
		}
		events = append(events, fetched...)
	}

	if len(results) > 0 {
		return results
	}

	if len(events) == 0 {
		return []Result{{
			Source:    job.sourceName,
			ProjectID: job.projectID,
			Duration:  time.Since(started),
			Skipped:   "no new events",
		}}
	}

	if err := store.WriteEvents(dataDir, job.sourceName, job.projectID, events); err != nil {
		return append(results, Result{
			Source:    job.sourceName,
			ProjectID: job.projectID,
			Duration:  time.Since(started),
			Error:     fmt.Sprintf("write: %v", err),
		})
	}

	return append(results, Result{
		Source:    job.sourceName,
		ProjectID: job.projectID,
		Records:   len(events),
		StartDate: opts.StartDate.Format("2006-01-02"),
		EndDate:   opts.EndDate.Format("2006-01-02"),
		Duration:  time.Since(started),
	})
}

func runMetricJob(ctx context.Context, dataDir string, job fetchJob, opts source.FetchOptions, started time.Time) []Result {
	var results []Result
	var records []source.Record
	contentByFilename := make(map[string][]source.ContentEntry)

	for _, src := range job.sources {
		fetched, err := src.Fetch(ctx, opts)
		if err != nil {
			results = append(results, Result{
				Source:    job.sourceName,
				ProjectID: job.projectID,
				Duration:  time.Since(started),
				Error:     err.Error(),
			})
			continue
		}
		if len(fetched) == 0 {
			continue
		}
		records = append(records, fetched...)
		if cp, ok := src.(source.ContentProvider); ok {
			if entries := cp.ContentEntries(); len(entries) > 0 {
				contentByFilename[cp.ContentFilename()] = append(contentByFilename[cp.ContentFilename()], entries...)
			}
		}
	}

	if len(results) > 0 {
		return results
	}

	if len(records) == 0 {
		return []Result{{
			Source:    job.sourceName,
			ProjectID: job.projectID,
			Duration:  time.Since(started),
			Skipped:   "no new records",
		}}
	}

	if err := store.WriteRecords(dataDir, job.sourceName, job.projectID, records); err != nil {
		return append(results, Result{
			Source:    job.sourceName,
			ProjectID: job.projectID,
			Duration:  time.Since(started),
			Error:     fmt.Sprintf("write: %v", err),
		})
	}

	for filename, entries := range contentByFilename {
		if err := store.WriteContent(dataDir, job.sourceName, job.projectID, filename, entries); err != nil {
			slog.Warn("write content failed", "source", job.sourceName, "project", job.projectID, "error", err)
		}
	}

	return append(results, Result{
		Source:    job.sourceName,
		ProjectID: job.projectID,
		Records:   len(records),
		StartDate: opts.StartDate.Format("2006-01-02"),
		EndDate:   opts.EndDate.Format("2006-01-02"),
		Duration:  time.Since(started),
	})
}

func All(ctx context.Context, cfg *config.Config, tokens Tokens, opts Options) ([]Result, error) {
	projects, err := selectedProjects(cfg, opts)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: 30 * time.Second}

	if tokens.LinkedIn != "" {
		auth.CheckLinkedInTokenExpiry(ctx, tokens.LinkedIn, os.Getenv("LINKEDIN_CLIENT_ID"), os.Getenv("LINKEDIN_CLIENT_SECRET"))
	}

	jobsByKey := make(map[jobKey]*fetchJob)
	var jobOrder []jobKey
	for id, proj := range projects {
		for _, repo := range proj.GitHubTraffic {
			addMetricJob(jobsByKey, &jobOrder, "github-traffic", id, &source.GitHubTraffic{Client: client, Token: tokens.GitHub, Repo: repo})
		}
		for _, repo := range proj.GitHubEvents {
			addEventJob(jobsByKey, &jobOrder, "github", id, &source.GitHubEvents{Client: client, Token: tokens.GitHub, Repo: repo})
		}
		for _, pkg := range proj.PyPI {
			addMetricJob(jobsByKey, &jobOrder, "pypi", id, &source.PyPI{Client: client, Package: pkg})
		}
		for _, pkg := range proj.CRAN {
			addMetricJob(jobsByKey, &jobOrder, "cran", id, &source.CRAN{Client: client, Package: pkg})
		}
		for _, formula := range proj.Homebrew {
			addMetricJob(jobsByKey, &jobOrder, "homebrew", id, &source.Homebrew{Client: client, Formula: formula})
		}
		if tokens.Plausible != "" {
			for _, site := range proj.Plausible {
				addMetricJob(jobsByKey, &jobOrder, "plausible", id, &source.Plausible{Client: client, APIKey: tokens.Plausible, SiteID: site})
			}
		}
		for _, ext := range proj.OpenVSX {
			addMetricJob(jobsByKey, &jobOrder, "openvsx", id, &source.OpenVSX{Client: client, ExtensionID: ext})
		}
		if tokens.YouTube != "" {
			for _, target := range proj.YouTube {
				addMetricJob(jobsByKey, &jobOrder, "youtube", id, &source.YouTube{Client: client, APIKey: tokens.YouTube, Target: target})
			}
		}
		if tokens.LinkedIn != "" {
			for _, target := range proj.LinkedIn {
				addMetricJob(jobsByKey, &jobOrder, "linkedin", id, &source.LinkedIn{Client: client, Token: tokens.LinkedIn, Target: target})
			}
		}
	}

	return runJobs(ctx, cfg, opts, orderedJobs(jobsByKey, jobOrder), 4)
}

func Source(ctx context.Context, cfg *config.Config, _ Tokens, sourceName string, opts Options, createSources func(id string, proj config.Project) []source.Source) ([]Result, error) {
	projects, err := selectedProjects(cfg, opts)
	if err != nil {
		return nil, err
	}

	var jobs []fetchJob

	for id, proj := range projects {
		sources := createSources(id, proj)
		if len(sources) > 0 {
			jobs = append(jobs, fetchJob{sourceName: sourceName, projectID: id, sources: sources})
		}
	}

	return runJobs(ctx, cfg, opts, jobs, 1)
}

func GitHub(ctx context.Context, cfg *config.Config, tokens Tokens, opts Options) ([]Result, error) {
	projects, err := selectedProjects(cfg, opts)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: 30 * time.Second}
	var jobs []fetchJob

	for id, proj := range projects {
		var eventSources []source.EventSource
		for _, repo := range proj.GitHubEvents {
			eventSources = append(eventSources, &source.GitHubEvents{Client: client, Token: tokens.GitHub, Repo: repo})
		}
		if len(eventSources) > 0 {
			jobs = append(jobs, fetchJob{sourceName: "github", projectID: id, eventSources: eventSources})
		}
	}

	return runJobs(ctx, cfg, opts, jobs, 1)
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
		return []Result{{Source: "plausible", Skipped: "PLAUSIBLE_TOKEN not set"}}, nil
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
		return []Result{{Source: "youtube", Skipped: "YOUTUBE_TOKEN not set"}}, nil
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

func LinkedIn(ctx context.Context, cfg *config.Config, tokens Tokens, opts Options) ([]Result, error) {
	if tokens.LinkedIn == "" {
		return []Result{{Source: "linkedin", Skipped: "LINKEDIN_TOKEN not set"}}, nil
	}
	auth.CheckLinkedInTokenExpiry(ctx, tokens.LinkedIn, os.Getenv("LINKEDIN_CLIENT_ID"), os.Getenv("LINKEDIN_CLIENT_SECRET"))
	client := &http.Client{Timeout: 30 * time.Second}
	return Source(ctx, cfg, tokens, "linkedin", opts, func(id string, p config.Project) []source.Source {
		var sources []source.Source
		for _, target := range p.LinkedIn {
			sources = append(sources, &source.LinkedIn{Client: client, Token: tokens.LinkedIn, Target: target})
		}
		return sources
	})
}
