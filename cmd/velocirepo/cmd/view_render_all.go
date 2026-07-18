package cmd

import (
	"fmt"
	"strings"

	"github.com/posit-dev/velocirepo/internal/views"
	"github.com/spf13/cobra"
)

func renderViewsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "render-views [prefix]",
		Short:   "Render all or filtered views",
		Args:    cobra.MaximumNArgs(1),
		GroupID: "view",
		RunE: func(cmd *cobra.Command, args []string) error {
			viewsDir := cfg.ViewsDir()

			rebuildDB()

			allViews, err := views.Discover(viewsDir)
			if err != nil {
				return err
			}

			var prefix string
			if len(args) > 0 {
				prefix = args[0]
			}

			var toRender []views.View
			for _, v := range allViews {
				if prefix == "" || v.Name == prefix || strings.HasPrefix(v.Name, prefix+"/") {
					toRender = append(toRender, v)
				}
			}

			if len(toRender) == 0 {
				if prefix != "" {
					return fmt.Errorf("no views matching prefix %q", prefix)
				}
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No views found. Use 'velocirepo add-view' to create one.")
				return nil
			}

			out := cmd.OutOrStdout()
			var rendered int
			for _, v := range toRender {
				if err := views.Render(v); err != nil {
					_, _ = fmt.Fprintf(out, "Error rendering '%s': %v\n", v.Name, err)
					continue
				}
				_, _ = fmt.Fprintf(out, "Rendered '%s'\n", v.Name)
				rendered++
			}

			_, _ = fmt.Fprintf(out, "Rendered %d view(s)\n", rendered)
			return nil
		},
	}

	return cmd
}
