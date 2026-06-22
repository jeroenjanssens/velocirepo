package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jeroenjanssens/velocirepo/internal/views"
	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/tw"
	"github.com/spf13/cobra"
)

func listViewsCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list-views",
		Short:   "List all views",
		GroupID: "view",
		RunE: func(cmd *cobra.Command, args []string) error {
			viewsDir := cfg.ViewsDir()
			allViews, err := views.Discover(viewsDir, cfg.Views.Items, cfg.ViewsSource())
			if err != nil {
				return err
			}

			if len(allViews) == 0 {
				fmt.Println("No views found. Use 'velocirepo add-view' to create one.")
				return nil
			}

			table := tablewriter.NewTable(os.Stdout,
				tablewriter.WithHeaderAutoFormat(tw.Off),
				tablewriter.WithHeaderAlignment(tw.AlignLeft),
				tablewriter.WithRowAlignment(tw.AlignLeft),
				tablewriter.WithRendition(tw.Rendition{
					Symbols: tw.NewSymbols(tw.StyleLight),
				}),
			)
			table.Header([]string{"NAME", "FRAMEWORK", "SOURCE", "OUTPUT"})

			for _, v := range allViews {
				output := v.Output
				if rel, err := filepath.Rel(".", output); err == nil {
					output = rel
				}
				table.Append([]string{v.Name, string(v.Framework), v.Source, output})
			}

			table.Render()
			return nil
		},
	}
}
