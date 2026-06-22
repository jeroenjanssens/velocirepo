package cmd

import (
	"fmt"

	"github.com/jeroenjanssens/velocirepo/internal/views"
	"github.com/spf13/cobra"
)

func serveViewCmd() *cobra.Command {
	var port string

	cmd := &cobra.Command{
		Use:     "serve-view <name>",
		Short:   "Start a dev server for a view",
		Long:    "Start the framework's live dev server (quarto preview, marimo edit, jupyter notebook).",
		Args:    cobra.ExactArgs(1),
		GroupID: "view",
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			viewsDir := cfg.ViewsDir()

			allViews, err := views.Discover(viewsDir, cfg.Views.Items, cfg.ViewsSource())
			if err != nil {
				return err
			}

			v, found := views.FindView(allViews, name)
			if !found {
				return fmt.Errorf("view %q not found in %s", name, viewsDir)
			}

			serveCmd, err := views.ServeCmd(v, port)
			if err != nil {
				return err
			}

			return serveCmd.Run()
		},
	}

	cmd.Flags().StringVar(&port, "port", "", "override port for dev server")

	return cmd
}
