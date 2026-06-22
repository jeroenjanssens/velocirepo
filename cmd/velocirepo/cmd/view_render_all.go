package cmd

import (
	"fmt"
	"strings"

	"github.com/jeroenjanssens/velocirepo/internal/views"
	"github.com/spf13/cobra"
)

func renderViewsCmd() *cobra.Command {
	var noExport bool

	cmd := &cobra.Command{
		Use:     "render-views [prefix]",
		Short:   "Render all or filtered views",
		Args:    cobra.MaximumNArgs(1),
		GroupID: "view",
		RunE: func(cmd *cobra.Command, args []string) error {
			viewsDir := cfg.ViewsDir()

			allViews, err := views.Discover(viewsDir, cfg.Views.Items, cfg.ViewsSource())
			if err != nil {
				return err
			}

			var prefix string
			if len(args) > 0 {
				prefix = args[0]
			}

			var toRender []views.View
			for _, v := range allViews {
				if prefix == "" || strings.HasPrefix(v.Name, prefix) {
					toRender = append(toRender, v)
				}
			}

			if len(toRender) == 0 {
				if prefix != "" {
					return fmt.Errorf("no views matching prefix %q", prefix)
				}
				fmt.Println("No views found. Use 'velocirepo add-view' to create one.")
				return nil
			}

			if !noExport && views.AnyUsesParquet(toRender) {
				if err := exportViewsData(viewsDir); err != nil {
					return fmt.Errorf("export data: %w", err)
				}
			}

			var rendered int
			for _, v := range toRender {
				if _, err := views.CheckRenderer(v.Framework, v.Venv); err != nil {
					fmt.Printf("Skipping '%s': %v\n", v.Name, err)
					continue
				}

				if err := views.Render(v); err != nil {
					fmt.Printf("Error rendering '%s': %v\n", v.Name, err)
					continue
				}

				fmt.Printf("Rendered '%s' → %s\n", v.Name, v.Output)
				rendered++
			}

			fmt.Printf("Rendered %d views\n", rendered)
			return nil
		},
	}

	cmd.Flags().BoolVar(&noExport, "no-export", false, "skip Parquet export (use existing _data/ files)")

	return cmd
}
