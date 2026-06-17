package cmd

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/jeroenjanssens/velocirepo/internal/source"
	"github.com/jeroenjanssens/velocirepo/internal/store"
	"github.com/spf13/cobra"
)

func fetchGitHubEventsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "github-events",
		Short: "Fetch raw GitHub events (stars, forks, issues, PRs, comments)",
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

			for id, proj := range projects {
				for _, repo := range proj.GitHubEvents {
					startDate, err := resolveStartDate(dataDir, "github-events", id)
					if err != nil {
						slog.Error("resolve start date", "project", id, "error", err)
						continue
					}

					if !startDate.Before(endDate.AddDate(0, 0, 1)) {
						slog.Info("up to date", "source", "github-events", "project", id)
						continue
					}

					slog.Info("fetching",
						"source", "github-events",
						"project", id,
						"start", startDate.Format("2006-01-02"),
						"end", endDate.Format("2006-01-02"),
					)

					src := &source.GitHubEvents{Client: client, Token: token, Repo: repo}
					events, err := src.FetchEvents(cmd.Context(), source.FetchOptions{
						ProjectID: id,
						StartDate: startDate,
						EndDate:   endDate,
					})
					if err != nil {
						slog.Error("fetch failed", "project", id, "error", err)
						continue
					}

					if len(events) == 0 {
						continue
					}

					if err := store.WriteGitHubEvents(dataDir, "github-events", id, events); err != nil {
						slog.Error("write failed", "project", id, "error", err)
						continue
					}

					slog.Info("wrote events", "source", "github-events", "project", id, "count", len(events))
				}
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
