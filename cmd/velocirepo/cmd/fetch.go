package cmd

import (
	"context"
	"fmt"

	"github.com/jeroenjanssens/velocirepo/internal/config"
	"github.com/jeroenjanssens/velocirepo/internal/fetch"
	"github.com/jeroenjanssens/velocirepo/internal/ui"
	"github.com/spf13/cobra"
)

var (
	fetchProject   string
	fetchStartDate string
	fetchEndDate   string
	noAggregate    bool
)

func addFetchFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&fetchProject, "project", "", "fetch only this project ID")
	cmd.Flags().StringVar(&fetchStartDate, "start-date", "", "start date (YYYY-MM-DD)")
	cmd.Flags().StringVar(&fetchEndDate, "end-date", "", "end date (YYYY-MM-DD, default: yesterday)")
	cmd.Flags().BoolVar(&noAggregate, "no-concatenate", false, "skip concatenation after fetch")
	cmd.GroupID = "fetch"
}

func fetchOpts() fetch.Options {
	return fetch.Options{
		Project:       fetchProject,
		StartDate:     fetchStartDate,
		EndDate:       fetchEndDate,
		NoConcatenate: noAggregate,
	}
}

func renderFetchResults(results []fetch.Result) {
	for _, r := range results {
		switch {
		case r.Error != "":
			ui.FetchError(r.Source, r.ProjectID, fmt.Errorf("%s", r.Error))
		case r.Skipped != "":
			ui.FetchSkip(r.Source, r.ProjectID, r.Skipped)
		default:
			ui.FetchDone(r.Source, r.ProjectID, r.Records, r.Duration)
		}
	}
}

type fetchSourceDef struct {
	use   string
	short string
	fn    func(context.Context, *config.Config, fetch.Tokens, fetch.Options) ([]fetch.Result, error)
}

var fetchSources = []fetchSourceDef{
	{"fetch-cran", "Fetch CRAN download statistics", fetch.CRAN},
	{"fetch-github", "Fetch GitHub events (stars, forks, issues, PRs)", fetch.GitHub},
	{"fetch-traffic", "Fetch GitHub traffic data (views and clones)", fetch.Traffic},
	{"fetch-homebrew", "Fetch Homebrew install counts", fetch.Homebrew},
	{"fetch-openvsx", "Fetch Open VSX extension metrics", fetch.OpenVSX},
	{"fetch-plausible", "Fetch Plausible analytics (pageviews, visitors, visits)", fetch.Plausible},
	{"fetch-pypi", "Fetch PyPI download statistics", fetch.PyPI},
	{"fetch-youtube", "Fetch YouTube metrics (views, likes, comments, subscribers)", fetch.YouTube},
}

func makeFetchCmd(def fetchSourceDef) *cobra.Command {
	cmd := &cobra.Command{
		Use:   def.use,
		Short: def.short,
		RunE: func(cmd *cobra.Command, args []string) error {
			results, err := def.fn(cmd.Context(), cfg, fetch.TokensFromEnv(), fetchOpts())
			if err != nil {
				return err
			}
			renderFetchResults(results)
			rebuildDB()
			return nil
		},
	}
	addFetchFlags(cmd)
	return cmd
}
