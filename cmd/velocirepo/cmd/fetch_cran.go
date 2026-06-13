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
			return runFetch(cmd, "cran", func(id string, p config.Project) source.Source {
				if p.CRAN == "" {
					return nil
				}
				return &source.CRAN{
					Client:  newHTTPClient(),
					Package: p.CRAN,
				}
			})
		},
	}
}
