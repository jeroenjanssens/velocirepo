package cmd

import (
	"fmt"

	"github.com/jeroenjanssens/velocirepo/internal/store"
	"github.com/jeroenjanssens/velocirepo/internal/views"
	"github.com/spf13/cobra"
)

func renderViewCmd() *cobra.Command {
	var noExport bool

	cmd := &cobra.Command{
		Use:     "render-view <name|directory>",
		Short:   "Render a view or all views in a directory",
		Args:    cobra.ExactArgs(1),
		GroupID: "view",
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			viewsDir := cfg.ViewsDir()

			allViews, err := views.Discover(viewsDir, cfg.Views.Items, cfg.ViewsSource())
			if err != nil {
				return err
			}

			toRender := views.FindViews(allViews, name)
			if len(toRender) == 0 {
				return fmt.Errorf("no views matching %q in %s", name, viewsDir)
			}

			if !noExport && views.AnyUsesParquet(toRender) {
				if err := exportViewsData(viewsDir); err != nil {
					return fmt.Errorf("export data: %w", err)
				}
			}

			for _, v := range toRender {
				if _, err := views.CheckRenderer(v.Framework, v.Venv); err != nil {
					fmt.Printf("Skipping '%s': %v\n", v.Name, err)
					continue
				}

				if err := views.Render(v); err != nil {
					return err
				}

				fmt.Printf("Rendered '%s' → %s\n", v.Name, v.Output)
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&noExport, "no-export", false, "skip Parquet export (use existing _data/ files)")

	return cmd
}

func exportViewsData(viewsDir string) error {
	dataDir := viewsDir + "/_data"
	_, err := store.Export(store.ExportOptions{
		DataDir:    cfg.DataDir(),
		OutDir:     dataDir,
		Format:     "parquet",
		Projects:   projectInfos(),
		Indicators: indicatorDefs(),
	})
	return err
}
