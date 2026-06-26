package cmd

import (
	"github.com/jeroenjanssens/velocirepo/internal/fetch"
	"github.com/spf13/cobra"
)

func fetchHomebrewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fetch-homebrew",
		Short: "Fetch Homebrew install counts",
		RunE: func(cmd *cobra.Command, args []string) error {
			results, err := fetch.Homebrew(cmd.Context(), cfg, fetch.TokensFromEnv(), fetchOpts())
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
