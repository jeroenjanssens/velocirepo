package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jeroenjanssens/velocirepo/internal/config"
	"github.com/spf13/cobra"
)

func projectRenameCmd() *cobra.Command {
	var noMoveData bool

	cmd := &cobra.Command{
		Use:   "rename <old-id> <new-id>",
		Short: "Rename a project's ID",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			oldID := args[0]
			newID := args[1]

			if !validIDRe.MatchString(newID) {
				return fmt.Errorf("invalid project ID %q: must be lowercase alphanumeric with hyphens", newID)
			}

			projects := cfg.ResolveProjects()
			if _, exists := projects[oldID]; !exists {
				return fmt.Errorf("project %q not found in config", oldID)
			}
			if _, exists := projects[newID]; exists {
				return fmt.Errorf("project %q already exists in config", newID)
			}

			cfgPath := cfgFilePath()
			if err := config.RenameSection(cfgPath, oldID, newID); err != nil {
				return err
			}

			if !noMoveData {
				dataDir := cfg.DataDir()
				sources := []string{"github", "github-traffic", "pypi", "cran", "homebrew", "plausible", "openvsx"}
				var moved []string
				for _, src := range sources {
					oldDir := filepath.Join(dataDir, src, oldID)
					newDir := filepath.Join(dataDir, src, newID)
					if _, err := os.Stat(oldDir); err == nil {
						if err := os.Rename(oldDir, newDir); err != nil {
							return fmt.Errorf("rename %s → %s: %w", oldDir, newDir, err)
						}
						moved = append(moved, src)
					}
				}
				if len(moved) > 0 {
					fmt.Fprintf(os.Stdout, "Renamed project '%s' → '%s' (moved data for: %s)\n", oldID, newID, joinComma(moved))
				} else {
					fmt.Fprintf(os.Stdout, "Renamed project '%s' → '%s' (no data directories found)\n", oldID, newID)
				}
			} else {
				fmt.Fprintf(os.Stdout, "Renamed project '%s' → '%s'\n", oldID, newID)
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&noMoveData, "no-move-data", false, "only rename in config, leave data directories as-is")

	return cmd
}

func joinComma(s []string) string {
	result := ""
	for i, v := range s {
		if i > 0 {
			result += ", "
		}
		result += v
	}
	return result
}
