package cmd

import (
	"github.com/jeroenjanssens/velocirepo/internal/config"
	"github.com/jeroenjanssens/velocirepo/internal/source"
	"github.com/spf13/cobra"
)

func fetchHomebrewCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "homebrew",
		Short: "Fetch Homebrew install counts",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runFetch(cmd, "homebrew", func(id string, proj config.Project) source.Source {
				if proj.Homebrew == "" {
					return nil
				}
				return &source.Homebrew{
					Client:  newHTTPClient(),
					Formula: proj.Homebrew,
				}
			})
		},
	}
}
