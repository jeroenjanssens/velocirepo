package cmd

import (
	"fmt"

	"github.com/posit-dev/velocirepo/internal/views"
	"github.com/spf13/cobra"
)

func renderViewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "render-view <name|directory>",
		Short:   "Render a view or all views in a directory",
		Args:    cobra.ExactArgs(1),
		GroupID: "view",
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			viewsDir := cfg.ViewsDir()

			rebuildDB()

			allViews, err := views.Discover(viewsDir)
			if err != nil {
				return err
			}

			toRender := views.FindViews(allViews, name)
			if len(toRender) == 0 {
				return fmt.Errorf("no views matching %q in %s", name, viewsDir)
			}

			for _, v := range toRender {
				if err := views.Render(v); err != nil {
					return err
				}
				fmt.Printf("Rendered '%s'\n", v.Name)
			}
			return nil
		},
	}

	return cmd
}
