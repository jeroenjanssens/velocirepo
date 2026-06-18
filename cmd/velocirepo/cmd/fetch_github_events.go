package cmd

import (
	"fmt"
	"time"

	"github.com/jeroenjanssens/velocirepo/internal/source"
	"github.com/jeroenjanssens/velocirepo/internal/store"
	"github.com/jeroenjanssens/velocirepo/internal/ui"
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
						ui.Errorf("github-events/%s: resolve start date: %v", id, err)
						continue
					}

					if !startDate.Before(endDate.AddDate(0, 0, 1)) {
						ui.Skip("github-events", id, "up to date")
						continue
					}

					ui.Progress("github-events", id, startDate.Format("2006-01-02")+" → "+endDate.Format("2006-01-02"))

					src := &source.GitHubEvents{Client: client, Token: token, Repo: repo}
					events, err := src.FetchEvents(cmd.Context(), source.FetchOptions{
						ProjectID: id,
						StartDate: startDate,
						EndDate:   endDate,
					})
					if err != nil {
						ui.Errorf("github-events/%s: %v", id, err)
						continue
					}

					if len(events) == 0 {
						continue
					}

					if err := store.WriteGitHubEvents(dataDir, "github-events", id, events); err != nil {
						ui.Errorf("github-events/%s write: %v", id, err)
						continue
					}

					ui.Done("github-events", id, len(events))
				}
			}

			if !noAggregate {
				if err := store.Aggregate(dataDir, time.Now().UTC()); err != nil {
					ui.Warnf("aggregation: %v", err)
				}
			}

			return nil
		},
	}
}
