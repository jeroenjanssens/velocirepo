package cmd

import (
	"github.com/jeroenjanssens/velocirepo/internal/fetch"
	"github.com/jeroenjanssens/velocirepo/internal/ui"
	"github.com/spf13/cobra"
)

func fetchAllCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fetch",
		Short: "Fetch from all configured sources",
		RunE: func(cmd *cobra.Command, args []string) error {
			results, err := fetch.All(cmd.Context(), cfg, fetch.TokensFromEnv(), fetchOpts())
			if err != nil {
				return err
			}
			renderFetchResults(results)
			rebuildDB()

			var successes, skips, errors int
			for _, r := range results {
				switch {
				case r.Error != "":
					errors++
				case r.Skipped != "":
					skips++
				default:
					successes++
				}
			}
			ui.Infof("Done: %d succeeded, %d skipped, %d failed", successes, skips, errors)
			return nil
		},
	}

	addFetchFlags(cmd)
	return cmd
}
