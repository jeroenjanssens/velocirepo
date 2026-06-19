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
		Use:   "github-events",
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
				for _, repo := range proj.GitHubEvents {
					startDate, err := resolveStartDate(dataDir, "github", id)
					if err != nil {
						ui.FetchError("github", id, fmt.Errorf("resolve start date: %w", err))
						continue
					}

					if !startDate.Before(endDate.AddDate(0, 0, 1)) {
						ui.FetchSkip("github", id, "already up to date")
						continue
					}

					dateRange := fmt.Sprintf("%s to %s", startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))
					ui.FetchStart("github", id, dateRange)
					started := time.Now()

					src := &source.GitHubEvents{Client: client, Token: token, Repo: repo}
					events, err := src.FetchEvents(cmd.Context(), source.FetchOptions{
						ProjectID: id,
						StartDate: startDate,
						EndDate:   endDate,
					})
					if err != nil {
						ui.FetchError("github", id, err)
						continue
					}

					if len(events) == 0 {
						ui.FetchSkip("github", id, "no new events")
						continue
					}

					if err := store.WriteGitHubEvents(dataDir, "github", id, events); err != nil {
						ui.FetchError("github", id, fmt.Errorf("write: %w", err))
						continue
					}

					ui.FetchDone("github", id, len(events), time.Since(started))
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
