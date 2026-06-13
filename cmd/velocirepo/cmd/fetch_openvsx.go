package cmd

import (
	"github.com/jeroenjanssens/velocirepo/internal/config"
	"github.com/jeroenjanssens/velocirepo/internal/source"
	"github.com/spf13/cobra"
)

func fetchOpenVSXCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "openvsx",
		Short: "Fetch Open VSX extension metrics",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runFetch(cmd, "openvsx", func(id string, p config.Project) source.Source {
				if p.OpenVSX == "" {
					return nil
				}
				return &source.OpenVSX{
					Client:      newHTTPClient(),
					ExtensionID: p.OpenVSX,
				}
			})
		},
	}
}
