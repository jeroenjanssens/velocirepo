package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/posit-dev/velocirepo/internal/config"
	"github.com/posit-dev/velocirepo/internal/store"
	"github.com/spf13/cobra"
)

func renameProjectCmd() *cobra.Command {
	var noMoveData bool

	cmd := &cobra.Command{
		Use:     "rename-project <old-id> <new-id>",
		Short:   "Rename a project's ID",
		Args:    cobra.ExactArgs(2),
		GroupID: "project",
		RunE: func(cmd *cobra.Command, args []string) error {
			oldID := args[0]
			newID := args[1]

			if !validIDRe.MatchString(newID) {
				return fmt.Errorf("invalid project ID %q: must be lowercase alphanumeric with hyphens", newID)
			}

			if _, err := cfg.GetProject(oldID); err != nil {
				return err
			}
			if _, err := cfg.GetProject(newID); err == nil {
				return fmt.Errorf("project %q already exists in config", newID)
			}

			var moved []string
			var dataMoves []projectDataMove
			if !noMoveData {
				var err error
				dataMoves, moved, err = moveProjectDataForRename(cfg.DataDir(), oldID, newID)
				if err != nil {
					return err
				}
			}

			cfgPath := cfgFilePath()
			if err := config.RenameSection(cfgPath, oldID, newID); err != nil {
				if rollbackErr := rollbackProjectDataMoves(dataMoves); rollbackErr != nil {
					return fmt.Errorf("%w (rollback data: %v)", err, rollbackErr)
				}
				return err
			}

			out := cmd.OutOrStdout()
			if !noMoveData {
				if len(moved) > 0 {
					_, _ = fmt.Fprintf(out, "Renamed project '%s' → '%s' (moved data for: %s)\n", oldID, newID, joinComma(moved))
				} else {
					_, _ = fmt.Fprintf(out, "Renamed project '%s' → '%s' (no data directories found)\n", oldID, newID)
				}
			} else {
				_, _ = fmt.Fprintf(out, "Renamed project '%s' → '%s'\n", oldID, newID)
			}

			rebuildDB()
			return nil
		},
	}

	cmd.Flags().BoolVar(&noMoveData, "no-move-data", false, "only rename in config, leave data directories as-is")

	return cmd
}

type projectDataMove struct {
	from  string
	to    string
	oldID string
	newID string
}

func moveProjectDataForRename(dataDir, oldID, newID string) ([]projectDataMove, []string, error) {
	for _, src := range config.SourceDirNames() {
		newDir := filepath.Join(dataDir, src, newID)
		if _, err := os.Stat(newDir); err == nil {
			return nil, nil, fmt.Errorf("target data directory already exists: %s", newDir)
		} else if !os.IsNotExist(err) {
			return nil, nil, err
		}
	}

	var moves []projectDataMove
	var moved []string
	for _, src := range config.SourceDirNames() {
		oldDir := filepath.Join(dataDir, src, oldID)
		newDir := filepath.Join(dataDir, src, newID)
		if _, err := os.Stat(oldDir); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return moves, moved, err
		}
		if err := os.MkdirAll(filepath.Dir(newDir), 0755); err != nil {
			_ = rollbackProjectDataMoves(moves)
			return nil, nil, err
		}
		if err := os.Rename(oldDir, newDir); err != nil {
			_ = rollbackProjectDataMoves(moves)
			return nil, nil, fmt.Errorf("rename %s → %s: %w", oldDir, newDir, err)
		}
		moves = append(moves, projectDataMove{from: oldDir, to: newDir, oldID: oldID, newID: newID})
		if err := store.RewriteProjectID(newDir, oldID, newID); err != nil {
			_ = rollbackProjectDataMoves(moves)
			return nil, nil, fmt.Errorf("rewrite project IDs in %s: %w", newDir, err)
		}
		moved = append(moved, src)
	}
	return moves, moved, nil
}

func rollbackProjectDataMoves(moves []projectDataMove) error {
	for i := len(moves) - 1; i >= 0; i-- {
		move := moves[i]
		if err := store.RewriteProjectID(move.to, move.newID, move.oldID); err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(move.from), 0755); err != nil {
			return err
		}
		if err := os.Rename(move.to, move.from); err != nil {
			return err
		}
	}
	return nil
}

func joinComma(s []string) string {
	return strings.Join(s, ", ")
}
