package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jeroenjanssens/velocirepo/internal/config"
	"github.com/spf13/cobra"
)

func removeProjectCmd() *cobra.Command {
	var force bool
	var deleteData bool

	cmd := &cobra.Command{
		Use:     "remove-project <id>",
		Aliases: []string{"rm-project"},
		Short:   "Remove a project from the config",
		Args:    cobra.ExactArgs(1),
		GroupID: "project",
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]

			if _, err := cfg.GetProject(id); err != nil {
				return err
			}

			if !force {
				if !isInteractive() {
					return fmt.Errorf("cannot prompt for confirmation (use --force)")
				}
				reader := bufio.NewReader(os.Stdin)
				ok, err := confirm(os.Stdout, reader, fmt.Sprintf("Remove project '%s' from config?", id))
				if err != nil {
					return err
				}
				if !ok {
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
				for _, src := range config.SourceDirNames() {
					dir := filepath.Join(dataDir, src, id)
					if _, err := os.Stat(dir); err == nil {
						_ = os.RemoveAll(dir)
					}
				}
				_, _ = fmt.Fprintf(os.Stdout, "Removed project '%s' and its data\n", id)
			} else {
				_, _ = fmt.Fprintf(os.Stdout, "Removed project '%s'\n", id)
			}

			rebuildDB()
			return nil
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "skip confirmation prompt")
	cmd.Flags().BoolVar(&deleteData, "delete-data", false, "also remove the project's data directories")

	return cmd
}
