package cmd

import (
	"github.com/jeroenjanssens/velocirepo/internal/fetch"
	"github.com/spf13/cobra"
)

func fetchPyPICmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fetch-pypi",
		Short: "Fetch PyPI download statistics",
		RunE: func(cmd *cobra.Command, args []string) error {
			results, err := fetch.PyPI(cmd.Context(), cfg, fetch.TokensFromEnv(), fetchOpts())
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
