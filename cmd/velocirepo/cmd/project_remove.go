package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jeroenjanssens/velocirepo/internal/config"
	"github.com/spf13/cobra"
)

func projectRemoveCmd() *cobra.Command {
	var force bool
	var deleteData bool

	cmd := &cobra.Command{
		Use:     "remove <id>",
		Aliases: []string{"rm"},
		Short:   "Remove a project from the config",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]

			projects := cfg.ResolveProjects()
			if _, exists := projects[id]; !exists {
				return fmt.Errorf("project %q not found in config", id)
			}

			if !force {
				if !isInteractive() {
					return fmt.Errorf("cannot prompt for confirmation (use --force)")
				}
				reader := bufio.NewReader(os.Stdin)
				if !confirm(os.Stdout, reader, fmt.Sprintf("Remove project '%s' from config?", id)) {
					fmt.Println("Cancelled.")
					return nil
				}
			}

			cfgPath := cfgFilePath()
			if err := config.RemoveProject(cfgPath, id); err != nil {
				return err
			}

			if deleteData {
				dataDir := cfg.DataDir()
				sources := []string{"github", "github-traffic", "pypi", "cran", "homebrew", "plausible", "openvsx"}
				for _, src := range sources {
					dir := filepath.Join(dataDir, src, id)
					if _, err := os.Stat(dir); err == nil {
						os.RemoveAll(dir)
					}
				}
				fmt.Fprintf(os.Stdout, "Removed project '%s' and its data\n", id)
			} else {
				fmt.Fprintf(os.Stdout, "Removed project '%s'\n", id)
			}

			return nil
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "skip confirmation prompt")
	cmd.Flags().BoolVar(&deleteData, "delete-data", false, "also remove the project's data directories")

	return cmd
}
