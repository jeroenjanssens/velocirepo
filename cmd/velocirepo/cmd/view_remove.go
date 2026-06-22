package cmd

import (
	"fmt"
	"os"

	"github.com/jeroenjanssens/velocirepo/internal/views"
	"github.com/spf13/cobra"
)

func removeViewCmd() *cobra.Command {
	var keepOutput bool

	cmd := &cobra.Command{
		Use:     "remove-view <name>",
		Short:   "Remove a view and its output",
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

			if err := os.Remove(v.Path); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("remove source: %w", err)
			}

			if !keepOutput {
				if err := os.Remove(v.Output); err != nil && !os.IsNotExist(err) {
					fmt.Fprintf(os.Stderr, "Warning: could not remove output %s: %v\n", v.Output, err)
				}
			}

			fmt.Printf("Removed view '%s'\n", name)
			return nil
		},
	}

	cmd.Flags().BoolVar(&keepOutput, "keep-output", false, "don't delete rendered output")

	return cmd
}
