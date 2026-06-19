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
					ui.Warnf("skipping plausible for %s: PLAUSIBLE_KEY not set", id)
				}
				for _, ext := range proj.OpenVSX {
					jobs = append(jobs, fetchJob{sourceName: "openvsx", projectID: id, src: &source.OpenVSX{Client: client, ExtensionID: ext}})
				}
				if ytKey := youtubeAPIKey(); ytKey != "" {
					for _, target := range proj.YouTube {
						jobs = append(jobs, fetchJob{sourceName: "youtube", projectID: id, src: &source.YouTube{Client: client, APIKey: ytKey, Target: target}})
					}
				} else if !proj.YouTube.IsEmpty() {
					ui.Warnf("skipping youtube for %s: YOUTUBE_API_KEY not set", id)
				}
			}

			var fetchErrors atomic.Int32
			g, ctx := errgroup.WithContext(cmd.Context())
			g.SetLimit(4)

			for _, job := range jobs {
				job := job
				g.Go(func() error {
					startDate, err := resolveStartDate(dataDir, job.sourceName, job.projectID)
					if err != nil {
						ui.Errorf("%s/%s: resolve start date: %v", job.sourceName, job.projectID, err)
						fetchErrors.Add(1)
						return nil
					}

					if !startDate.Before(endDate.AddDate(0, 0, 1)) {
						ui.Skip(job.sourceName, job.projectID, "up to date")
						return nil
					}

					ui.Progress(job.sourceName, job.projectID, "")

					if job.eventSrc != nil {
						events, err := job.eventSrc.FetchEvents(ctx, source.FetchOptions{
							ProjectID: job.projectID,
							StartDate: startDate,
							EndDate:   endDate,
						})
						if err != nil {
							ui.Errorf("%s/%s: %v", job.sourceName, job.projectID, err)
							fetchErrors.Add(1)
							return nil
						}

						if len(events) == 0 {
							return nil
						}

						if err := store.WriteGitHubEvents(dataDir, job.sourceName, job.projectID, events); err != nil {
							ui.Errorf("%s/%s write: %v", job.sourceName, job.projectID, err)
							fetchErrors.Add(1)
						} else {
							ui.Done(job.sourceName, job.projectID, len(events))
						}
					} else {
						records, err := job.src.Fetch(ctx, source.FetchOptions{
							ProjectID: job.projectID,
							StartDate: startDate,
							EndDate:   endDate,
						})
						if err != nil {
							ui.Errorf("%s/%s: %v", job.sourceName, job.projectID, err)
							fetchErrors.Add(1)
							return nil
						}

						if len(records) == 0 {
							return nil
						}

						if err := store.WriteRecords(dataDir, job.sourceName, job.projectID, records); err != nil {
							ui.Errorf("%s/%s write: %v", job.sourceName, job.projectID, err)
							fetchErrors.Add(1)
						} else {
							if yt, ok := job.src.(*source.YouTube); ok {
								if entries := yt.IndexEntries(); len(entries) > 0 {
									if err := store.WriteYouTubeIndex(dataDir, job.projectID, entries); err != nil {
										ui.Errorf("%s/%s index: %v", job.sourceName, job.projectID, err)
									}
								}
							}
							ui.Done(job.sourceName, job.projectID, len(records))
						}
					}

					return nil
				})
			}

			g.Wait()

			if !noAggregate {
				if err := store.Aggregate(dataDir, time.Now().UTC()); err != nil {
					ui.Warnf("aggregation: %v", err)
				}
			}

			if n := fetchErrors.Load(); n > 0 {
				return fmt.Errorf("%d source(s) failed to fetch", n)
			}
			return nil
		},
	}
}
