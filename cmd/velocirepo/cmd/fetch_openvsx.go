package cmd

import (
	"github.com/jeroenjanssens/velocirepo/internal/fetch"
	"github.com/spf13/cobra"
)

func fetchOpenVSXCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fetch-openvsx",
		Short: "Fetch Open VSX extension metrics",
		RunE: func(cmd *cobra.Command, args []string) error {
			results, err := fetch.OpenVSX(cmd.Context(), cfg, fetch.TokensFromEnv(), fetchOpts())
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
