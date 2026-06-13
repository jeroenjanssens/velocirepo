package cmd

import (
	"github.com/spf13/cobra"
)

func ciCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ci",
		Short: "CI/CD helpers",
	}

	cmd.AddCommand(ciInstallCmd())
	return cmd
}
