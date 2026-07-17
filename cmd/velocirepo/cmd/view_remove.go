package cmd

import (
	"bufio"
	"fmt"
	"os"

	"github.com/posit-dev/velocirepo/internal/views"
	"github.com/spf13/cobra"
)

func removeViewCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:     "remove-view <name>",
		Short:   "Remove a view directory",
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

			if !force {
				reader := bufio.NewReader(os.Stdin)
				ok, err := confirm(os.Stdout, reader, fmt.Sprintf("Remove view '%s' at %s?", name, v.Dir))
				if err != nil {
					return err
				}
				if !ok {
					return nil
				}
			}

			if err := os.RemoveAll(v.Dir); err != nil {
				return fmt.Errorf("remove view directory: %w", err)
			}

			fmt.Printf("Removed view '%s'\n", name)
			return nil
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "skip confirmation prompt")

	return cmd
}
