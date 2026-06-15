package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/jeroenjanssens/velocirepo/internal/source"
	"github.com/jeroenjanssens/velocirepo/internal/store"
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
			}

			var jobs []fetchJob
			for id, proj := range projects {
				for _, repo := range proj.GitHub {
					jobs = append(jobs, fetchJob{"github", id, &source.GitHub{Client: client, Token: token, Repo: repo}})
				}
				for _, repo := range proj.GitHubTraffic {
					jobs = append(jobs, fetchJob{"github-traffic", id, &source.GitHubTraffic{Client: client, Token: token, Repo: repo}})
				}
				for _, pkg := range proj.PyPI {
					jobs = append(jobs, fetchJob{"pypi", id, &source.PyPI{Client: client, Package: pkg}})
				}
				for _, pkg := range proj.CRAN {
					jobs = append(jobs, fetchJob{"cran", id, &source.CRAN{Client: client, Package: pkg}})
				}
				for _, formula := range proj.Homebrew {
					jobs = append(jobs, fetchJob{"homebrew", id, &source.Homebrew{Client: client, Formula: formula}})
				}
				if pKey != "" {
					for _, site := range proj.Plausible {
						jobs = append(jobs, fetchJob{"plausible", id, &source.Plausible{Client: client, APIKey: pKey, SiteID: site}})
					}
				}
				for _, ext := range proj.OpenVSX {
					jobs = append(jobs, fetchJob{"openvsx", id, &source.OpenVSX{Client: client, ExtensionID: ext}})
				}
			}

			g, ctx := errgroup.WithContext(cmd.Context())
			g.SetLimit(4)

			for _, job := range jobs {
				job := job
				g.Go(func() error {
					startDate, err := resolveStartDate(dataDir, job.sourceName, job.projectID)
					if err != nil {
						slog.Error("resolve start date", "source", job.sourceName, "project", job.projectID, "error", err)
						return nil
					}

					if !startDate.Before(endDate.AddDate(0, 0, 1)) {
						slog.Info("up to date", "source", job.sourceName, "project", job.projectID)
						return nil
					}

					slog.Info("fetching", "source", job.sourceName, "project", job.projectID)

					records, err := job.src.Fetch(ctx, source.FetchOptions{
						ProjectID: job.projectID,
						StartDate: startDate,
						EndDate:   endDate,
					})
					if err != nil {
						slog.Error("fetch failed", "source", job.sourceName, "project", job.projectID, "error", err)
						return nil
					}

					if len(records) == 0 {
						return nil
					}

					if err := store.WriteRecords(dataDir, job.sourceName, job.projectID, records); err != nil {
						slog.Error("write failed", "source", job.sourceName, "project", job.projectID, "error", err)
					} else {
						slog.Info("wrote records", "source", job.sourceName, "project", job.projectID, "count", len(records))
					}

					return nil
				})
			}

			if err := g.Wait(); err != nil {
				fmt.Fprintf(os.Stderr, "fetch all: %v\n", err)
			}

			if !noAggregate {
				if err := store.Aggregate(dataDir, time.Now().UTC()); err != nil {
					slog.Warn("aggregation failed", "error", err)
				}
			}

			return nil
		},
	}
}
