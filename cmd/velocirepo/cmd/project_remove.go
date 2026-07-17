package cmd

import (
	"bufio"
	"fmt"
	"os"

	"github.com/posit-dev/velocirepo/internal/config"
	"github.com/posit-dev/velocirepo/internal/projectdata"
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
			if err := removeProjectConfigAndData(cfgPath, cfg.DataDir(), id, deleteData, config.RemoveProject); err != nil {
				return err
			}

			if deleteData {
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

func removeProjectConfigAndData(cfgPath, dataDir, id string, deleteData bool, removeProject func(string, string) error) error {
	var trashRoot string
	var dataMoves []projectdata.Move

	if deleteData {
		var err error
		trashRoot, dataMoves, err = projectdata.TrashProjectDirs(dataDir, id)
		if err != nil {
			return fmt.Errorf("remove data: %w", err)
		}
	}

	if err := removeProject(cfgPath, id); err != nil {
		if len(dataMoves) > 0 {
			if rollbackErr := projectdata.RollbackMoves(dataMoves); rollbackErr != nil {
				return fmt.Errorf("%w (rollback data: %v)", err, rollbackErr)
			}
			_ = os.RemoveAll(trashRoot)
		}
		return err
	}

	if trashRoot != "" {
		if err := os.RemoveAll(trashRoot); err != nil {
			return fmt.Errorf("cleanup removed data: %w", err)
		}
	}

	return nil
}
