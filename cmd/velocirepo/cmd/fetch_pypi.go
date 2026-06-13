package cmd

import (
	"github.com/jeroenjanssens/velocirepo/internal/config"
	"github.com/jeroenjanssens/velocirepo/internal/source"
	"github.com/spf13/cobra"
)

func fetchPyPICmd() *cobra.Command {
	return &cobra.Command{
		Use:   "pypi",
		Short: "Fetch PyPI download statistics",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runFetch(cmd, "pypi", func(id string, p config.Project) source.Source {
				if p.PyPI == "" {
					return nil
				}
				return &source.PyPI{
					Client:  newHTTPClient(),
					Package: p.PyPI,
				}
			})
		},
	}
}
