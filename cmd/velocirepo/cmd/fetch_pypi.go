package cmd

import (
	"github.com/jeroenjanssens/velocirepo/internal/config"
	"github.com/jeroenjanssens/velocirepo/internal/source"
	"github.com/spf13/cobra"
)

func fetchPyPICmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fetch-pypi",
		Short: "Fetch PyPI download statistics",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runFetchMulti(cmd, "pypi", func(id string, p config.Project) []source.Source {
				var sources []source.Source
				for _, pkg := range p.PyPI {
					sources = append(sources, &source.PyPI{
						Client:  newHTTPClient(),
						Package: pkg,
					})
				}
				return sources
			})
		},
	}

	addFetchFlags(cmd)
	return cmd
}
