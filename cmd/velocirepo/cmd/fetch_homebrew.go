package cmd

import (
	"github.com/jeroenjanssens/velocirepo/internal/config"
	"github.com/jeroenjanssens/velocirepo/internal/source"
	"github.com/spf13/cobra"
)

func fetchHomebrewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fetch-homebrew",
		Short: "Fetch Homebrew install counts",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runFetchMulti(cmd, "homebrew", func(id string, proj config.Project) []source.Source {
				var sources []source.Source
				for _, formula := range proj.Homebrew {
					sources = append(sources, &source.Homebrew{
						Client:  newHTTPClient(),
						Formula: formula,
					})
				}
				return sources
			})
		},
	}

	addFetchFlags(cmd)
	return cmd
}
