package cmd

import (
	"fmt"

	"github.com/jeroenjanssens/velocirepo/internal/config"
	"github.com/jeroenjanssens/velocirepo/internal/source"
	"github.com/spf13/cobra"
)

func fetchPlausibleCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fetch-plausible",
		Short: "Fetch Plausible analytics (pageviews, visitors, visits)",
		RunE: func(cmd *cobra.Command, args []string) error {
			key := plausibleKey()
			if key == "" {
				return fmt.Errorf("PLAUSIBLE_TOKEN environment variable is required")
			}
			return runFetchMulti(cmd, "plausible", func(id string, p config.Project) []source.Source {
				var sources []source.Source
				for _, site := range p.Plausible {
					sources = append(sources, &source.Plausible{
						Client: newHTTPClient(),
						APIKey: key,
						SiteID: site,
					})
				}
				return sources
			})
		},
	}

	addFetchFlags(cmd)
	return cmd
}
