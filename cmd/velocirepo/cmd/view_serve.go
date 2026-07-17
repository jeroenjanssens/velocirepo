package cmd

import (
	"fmt"

	"github.com/posit-dev/velocirepo/internal/views"
	"github.com/spf13/cobra"
)

func serveViewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "serve-view <name>",
		Short:   "Start a dev server for a view",
		Long:    "Start the view's serve.sh script for live development.",
		Args:    cobra.ExactArgs(1),
		GroupID: "view",
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			viewsDir := cfg.ViewsDir()

			allViews, err := views.Discover(viewsDir)
			if err != nil {
				return err
			}

			v, found := views.FindView(allViews, name)
			if !found {
				return fmt.Errorf("view %q not found in %s", name, viewsDir)
			}

			serveCmd, err := views.ServeCmd(v)
			if err != nil {
				return err
			}

			return serveCmd.Run()
		},
	}

	return cmd
}
