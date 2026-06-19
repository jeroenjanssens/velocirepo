package cmd

import (
	"fmt"
	"sync/atomic"
	"time"

	"github.com/jeroenjanssens/velocirepo/internal/source"
	"github.com/jeroenjanssens/velocirepo/internal/store"
	"github.com/jeroenjanssens/velocirepo/internal/ui"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

func fetchAllCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "all",
		Short: "Fetch from all configured sources",
		RunE: func(cmd *cobra.Command, args []string) error {
			projects := cfg.ResolveProjects()
			if projects == nil {
				return fmt.Errorf("no projects configured")
			}

			projects = filterProjects(projects)
			if projects == nil {
				return fmt.Errorf("project %q not found in config", fetchProject)
			}

			endDate, err := resolveEndDate()
			if err != nil {
				return fmt.Errorf("parse end date: %w", err)
			}

			dataDir := cfg.DataDir()
			client := newHTTPClient()
			token := githubToken()
			pKey := plausibleKey()

			type fetchJob struct {
				sourceName string
				projectID  string
				src        source.Source
				eventSrc   source.GitHubEventSource
			}

			var jobs []fetchJob
			for id, proj := range projects {
					for _, repo := range proj.GitHubTraffic {
					jobs = append(jobs, fetchJob{sourceName: "github-traffic", projectID: id, src: &source.GitHubTraffic{Client: client, Token: token, Repo: repo}})
				}
				for _, repo := range proj.GitHubEvents {
					jobs = append(jobs, fetchJob{sourceName: "github", projectID: id, eventSrc: &source.GitHubEvents{Client: client, Token: token, Repo: repo}})
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
				if pKey != "" {
					for _, site := range proj.Plausible {
						jobs = append(jobs, fetchJob{sourceName: "plausible", projectID: id, src: &source.Plausible{Client: client, APIKey: pKey, SiteID: site}})
					}
				} else if !proj.Plausible.IsEmpty() {
					ui.Warnf("skipping plausible for %s: PLAUSIBLE_TOKEN not set", id)
				}
				for _, ext := range proj.OpenVSX {
					jobs = append(jobs, fetchJob{sourceName: "openvsx", projectID: id, src: &source.OpenVSX{Client: client, ExtensionID: ext}})
				}
				if ytKey := youtubeAPIKey(); ytKey != "" {
					for _, target := range proj.YouTube {
						jobs = append(jobs, fetchJob{sourceName: "youtube", projectID: id, src: &source.YouTube{Client: client, APIKey: ytKey, Target: target}})
					}
				} else if !proj.YouTube.IsEmpty() {
					ui.Warnf("skipping youtube for %s: YOUTUBE_TOKEN not set", id)
				}
			}

			ui.Infof("Starting fetch for %d job(s), end date: %s", len(jobs), endDate.Format("2006-01-02"))

			var fetchErrors atomic.Int32
			g, ctx := errgroup.WithContext(cmd.Context())
			g.SetLimit(4)

			for _, job := range jobs {
				job := job
				g.Go(func() error {
					startDate, err := resolveStartDate(dataDir, job.sourceName, job.projectID)
					if err != nil {
						ui.FetchError(job.sourceName, job.projectID, fmt.Errorf("resolve start date: %w", err))
						fetchErrors.Add(1)
						return nil
					}

					if !startDate.Before(endDate.AddDate(0, 0, 1)) {
						ui.FetchSkip(job.sourceName, job.projectID, "already up to date")
						return nil
					}

					dateRange := fmt.Sprintf("%s to %s", startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))
					ui.FetchStart(job.sourceName, job.projectID, dateRange)
					started := time.Now()

					if job.eventSrc != nil {
						events, err := job.eventSrc.FetchEvents(ctx, source.FetchOptions{
							ProjectID: job.projectID,
							StartDate: startDate,
							EndDate:   endDate,
						})
						if err != nil {
							ui.FetchError(job.sourceName, job.projectID, err)
							fetchErrors.Add(1)
							return nil
						}

						if len(events) == 0 {
							ui.FetchSkip(job.sourceName, job.projectID, "no new events")
							return nil
						}

						if err := store.WriteGitHubEvents(dataDir, job.sourceName, job.projectID, events); err != nil {
							ui.FetchError(job.sourceName, job.projectID, fmt.Errorf("write: %w", err))
							fetchErrors.Add(1)
						} else {
							ui.FetchDone(job.sourceName, job.projectID, len(events), time.Since(started))
						}
					} else {
						records, err := job.src.Fetch(ctx, source.FetchOptions{
							ProjectID: job.projectID,
							StartDate: startDate,
							EndDate:   endDate,
						})
						if err != nil {
							ui.FetchError(job.sourceName, job.projectID, err)
							fetchErrors.Add(1)
							return nil
						}

						if len(records) == 0 {
							ui.FetchSkip(job.sourceName, job.projectID, "no new records")
							return nil
						}

						if err := store.WriteRecords(dataDir, job.sourceName, job.projectID, records); err != nil {
							ui.FetchError(job.sourceName, job.projectID, fmt.Errorf("write: %w", err))
							fetchErrors.Add(1)
						} else {
							if yt, ok := job.src.(*source.YouTube); ok {
								if entries := yt.IndexEntries(); len(entries) > 0 {
									if err := store.WriteYouTubeIndex(dataDir, job.projectID, entries); err != nil {
										ui.FetchWarn(job.sourceName, job.projectID, fmt.Sprintf("index write: %v", err))
									}
								}
							}
							ui.FetchDone(job.sourceName, job.projectID, len(records), time.Since(started))
						}
					}

					return nil
				})
			}

			g.Wait()

			if !noAggregate {
				ui.Infof("Running concatenation...")
				concatStart := time.Now()
				if err := store.Aggregate(dataDir, time.Now().UTC()); err != nil {
					ui.Warnf("concatenation: %v", err)
				} else {
					ui.Infof("Concatenation complete in %s", time.Since(concatStart).Round(time.Millisecond))
				}
			}

			if n := fetchErrors.Load(); n > 0 {
				return fmt.Errorf("%d source(s) failed to fetch", n)
			}
			return nil
		},
	}
}
