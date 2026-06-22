package cmd

import (
	"fmt"

	"github.com/jeroenjanssens/velocirepo/internal/views"
	"github.com/spf13/cobra"
)

func showViewCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "show-view <name>",
		Short:   "Show details about a view",
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

			fmt.Printf("Name:       %s\n", v.Name)
			fmt.Printf("Framework:  %s\n", v.Framework)
			fmt.Printf("Source:     %s\n", v.Source)
			fmt.Printf("File:       %s\n", v.Path)
			fmt.Printf("Output:     %s\n", v.Output)

			if v.Venv != "" {
				fmt.Printf("Venv:       %s\n", v.Venv)
			} else {
				fmt.Printf("Venv:       (none)\n")
			}

			ver, err := views.CheckRenderer(v.Framework, v.Venv)
			if err != nil {
				fmt.Printf("Renderer:   not found (%v)\n", err)
			} else {
				fmt.Printf("Renderer:   %s %s\n", v.Framework, ver)
			}

			return nil
		},
	}
}
