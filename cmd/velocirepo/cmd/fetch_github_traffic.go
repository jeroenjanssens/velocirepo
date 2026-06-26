package cmd

import (
	"github.com/jeroenjanssens/velocirepo/internal/fetch"
	"github.com/spf13/cobra"
)

func fetchGitHubTrafficCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fetch-traffic",
		Short: "Fetch GitHub traffic data (views and clones)",
		RunE: func(cmd *cobra.Command, args []string) error {
			results, err := fetch.Traffic(cmd.Context(), cfg, fetch.TokensFromEnv(), fetchOpts())
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
