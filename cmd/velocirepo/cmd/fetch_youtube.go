package cmd

import (
	"github.com/jeroenjanssens/velocirepo/internal/fetch"
	"github.com/spf13/cobra"
)

func fetchYouTubeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fetch-youtube",
		Short: "Fetch YouTube metrics (views, likes, comments, subscribers)",
		RunE: func(cmd *cobra.Command, args []string) error {
			results, err := fetch.YouTube(cmd.Context(), cfg, fetch.TokensFromEnv(), fetchOpts())
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
