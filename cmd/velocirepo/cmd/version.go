package cmd

import (
	"fmt"

	"github.com/posit-dev/velocirepo/internal/version"
	"github.com/spf13/cobra"
)

func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the version",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "velocirepo %s (commit: %s, built: %s)\n", version.Version, version.Commit, version.Date)
			return nil
		},
	}
}
