package cmd

import (
	"fmt"

	"github.com/jeroenjanssens/velocirepo/internal/config"
	"github.com/jeroenjanssens/velocirepo/internal/source"
	"github.com/spf13/cobra"
)

func fetchPlausibleCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "plausible",
		Short: "Fetch Plausible analytics (pageviews, visitors, visits)",
		RunE: func(cmd *cobra.Command, args []string) error {
			key := plausibleKey()
			if key == "" {
				return fmt.Errorf("PLAUSIBLE_KEY environment variable is required")
			}
			return runFetch(cmd, "plausible", func(id string, p config.Project) source.Source {
				if p.Plausible == "" {
					return nil
				}
				return &source.Plausible{
					Client: newHTTPClient(),
					APIKey: key,
					SiteID: p.Plausible,
				}
			})
		},
	}
}
