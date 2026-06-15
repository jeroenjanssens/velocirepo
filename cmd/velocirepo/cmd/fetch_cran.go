package cmd

import (
	"github.com/jeroenjanssens/velocirepo/internal/config"
	"github.com/jeroenjanssens/velocirepo/internal/source"
	"github.com/spf13/cobra"
)

func fetchCRANCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "cran",
		Short: "Fetch CRAN download statistics",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runFetchMulti(cmd, "cran", func(id string, p config.Project) []source.Source {
				var sources []source.Source
				for _, pkg := range p.CRAN {
					sources = append(sources, &source.CRAN{
						Client:  newHTTPClient(),
						Package: pkg,
					})
				}
				return sources
			})
		},
	}
}
