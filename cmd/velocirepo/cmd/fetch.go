package cmd

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/jeroenjanssens/velocirepo/internal/config"
	"github.com/jeroenjanssens/velocirepo/internal/source"
	"github.com/jeroenjanssens/velocirepo/internal/store"
	"github.com/jeroenjanssens/velocirepo/internal/ui"
	"github.com/spf13/cobra"
)

var (
	fetchProject   string
	fetchStartDate string
	fetchEndDate   string
	noAggregate    bool
)

func fetchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fetch",
		Short: "Fetch metrics from data sources",
		Long:  "Fetch metrics from one or all configured data sources.",
	}

	cmd.PersistentFlags().StringVar(&fetchProject, "project", "", "fetch only this project ID")
	cmd.PersistentFlags().StringVar(&fetchStartDate, "start-date", "", "start date (YYYY-MM-DD)")
	cmd.PersistentFlags().StringVar(&fetchEndDate, "end-date", "", "end date (YYYY-MM-DD, default: yesterday)")
	cmd.PersistentFlags().BoolVar(&noAggregate, "no-aggregate", false, "skip aggregation after fetch")

	cmd.AddCommand(fetchGitHubCmd())
	cmd.AddCommand(fetchGitHubTrafficCmd())
	cmd.AddCommand(fetchGitHubEventsCmd())
	cmd.AddCommand(fetchPyPICmd())
	cmd.AddCommand(fetchCRANCmd())
	cmd.AddCommand(fetchHomebrewCmd())
	cmd.AddCommand(fetchPlausibleCmd())
	cmd.AddCommand(fetchOpenVSXCmd())
	cmd.AddCommand(fetchAllCmd())

	return cmd
}

func resolveEndDate() (time.Time, error) {
	if fetchEndDate != "" {
		return time.Parse("2006-01-02", fetchEndDate)
	}
	if cfg.Settings.EndDate == "yesterday" || cfg.Settings.EndDate == "" {
		return time.Now().UTC().AddDate(0, 0, -1).Truncate(24 * time.Hour), nil
	}
	return time.Parse("2006-01-02", cfg.Settings.EndDate)
}

func resolveStartDate(dataDir, sourceName, projectID string) (time.Time, error) {
	if fetchStartDate != "" {
		return time.Parse("2006-01-02", fetchStartDate)
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

	// GitHub Traffic API only retains 14 days of history
	if sourceName == "github-traffic" {
		earliest := time.Now().UTC().AddDate(0, 0, -14).Truncate(24 * time.Hour)
		if start.Before(earliest) {
			start = earliest
		}
	}

	return start, nil
}

func filterProjects(projects map[string]config.Project) map[string]config.Project {
	if fetchProject == "" {
		return projects
	}
	p, ok := projects[fetchProject]
	if !ok {
		return nil
	}
	return map[string]config.Project{fetchProject: p}
}

func runFetchMulti(cmd *cobra.Command, sourceName string, createSources func(projectID string, project config.Project) []source.Source) error {
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

	for id, proj := range projects {
		sources := createSources(id, proj)
		if len(sources) == 0 {
			continue
		}

		startDate, err := resolveStartDate(dataDir, sourceName, id)
		if err != nil {
			ui.Errorf("resolve start date for %s: %v", id, err)
			continue
		}

		if !startDate.Before(endDate.AddDate(0, 0, 1)) {
			ui.Skip(sourceName, id, "up to date")
			continue
		}

		for _, src := range sources {
			ui.Progress(sourceName, id, startDate.Format("2006-01-02")+" → "+endDate.Format("2006-01-02"))

			records, err := src.Fetch(cmd.Context(), source.FetchOptions{
				ProjectID: id,
				StartDate: startDate,
				EndDate:   endDate,
			})
			if err != nil {
				ui.Errorf("%s/%s: %v", sourceName, id, err)
				continue
			}

			if len(records) == 0 {
				ui.Skip(sourceName, id, "no data")
				continue
			}

			if err := store.WriteRecords(dataDir, sourceName, id, records); err != nil {
				ui.Errorf("%s/%s write: %v", sourceName, id, err)
				continue
			}

			ui.Done(sourceName, id, len(records))
		}
	}

	if !noAggregate {
		if err := store.Aggregate(dataDir, time.Now().UTC()); err != nil {
			ui.Warnf("aggregation: %v", err)
		}
	}

	return nil
}

func newHTTPClient() *http.Client {
	return &http.Client{Timeout: 30 * time.Second}
}

func githubToken() string {
	return os.Getenv("GITHUB_TOKEN")
}

func plausibleKey() string {
	return os.Getenv("PLAUSIBLE_KEY")
}
