package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/posit-dev/velocirepo/internal/views"
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

			allViews, err := views.Discover(viewsDir)
			if err != nil {
				return err
			}

			v, found := views.FindView(allViews, name)
			if !found {
				return fmt.Errorf("view %q not found in %s", name, viewsDir)
			}

			out := cmd.OutOrStdout()
			_, _ = fmt.Fprintf(out, "Name:  %s\n", v.Name)
			_, _ = fmt.Fprintf(out, "Dir:   %s\n", v.Dir)

			entries, err := os.ReadDir(v.Dir)
			if err == nil {
				var files []string
				for _, e := range entries {
					if !e.IsDir() {
						files = append(files, e.Name())
					}
				}
				if len(files) > 0 {
					_, _ = fmt.Fprintf(out, "Files: %s\n", strings.Join(files, ", "))
				}
			}

			renderScript := filepath.Join(v.Dir, "render.sh")
			if _, err := os.Stat(renderScript); err == nil {
				data, _ := os.ReadFile(renderScript)
				_, _ = fmt.Fprintf(out, "\nrender.sh:\n%s", string(data))
			}

			return nil
		},
	}
}
