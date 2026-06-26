package cmd

import (
	"github.com/jeroenjanssens/velocirepo/internal/fetch"
	"github.com/spf13/cobra"
)

func fetchGitHubCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fetch-github",
		Short: "Fetch GitHub events (stars, forks, issues, PRs)",
		RunE: func(cmd *cobra.Command, args []string) error {
			results, err := fetch.GitHub(cmd.Context(), cfg, fetch.TokensFromEnv(), fetchOpts())
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
