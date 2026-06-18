package cmd

import (
	"fmt"
	"time"

	"github.com/jeroenjanssens/velocirepo/internal/source"
	"github.com/jeroenjanssens/velocirepo/internal/store"
	"github.com/jeroenjanssens/velocirepo/internal/ui"
	"github.com/spf13/cobra"
)

func fetchGitHubCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "github",
		Short: "Fetch GitHub events (stars, forks, issues, PRs)",
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
				for _, repo := range proj.GitHub {
					startDate, err := resolveStartDate(dataDir, "github", id)
					if err != nil {
						ui.Errorf("github/%s: resolve start date: %v", id, err)
						continue
					}

					if !startDate.Before(endDate.AddDate(0, 0, 1)) {
						ui.Skip("github", id, "up to date")
						continue
					}

					ui.Progress("github", id, startDate.Format("2006-01-02")+" → "+endDate.Format("2006-01-02"))

					src := &source.GitHubEvents{Client: client, Token: token, Repo: repo}
					events, err := src.FetchEvents(cmd.Context(), source.FetchOptions{
						ProjectID: id,
						StartDate: startDate,
						EndDate:   endDate,
					})
					if err != nil {
						ui.Errorf("github/%s: %v", id, err)
						continue
					}

					if len(events) == 0 {
						continue
					}

					if err := store.WriteGitHubEvents(dataDir, "github", id, events); err != nil {
						ui.Errorf("github/%s write: %v", id, err)
						continue
					}

					ui.Done("github", id, len(events))
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
