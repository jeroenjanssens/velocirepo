package cmd

import (
	"fmt"

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
