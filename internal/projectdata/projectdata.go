package projectdata

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jeroenjanssens/velocirepo/internal/config"
	"github.com/jeroenjanssens/velocirepo/internal/store"
)

type Move struct {
	from  string
	to    string
	oldID string
	newID string
}

func MoveProjectDirs(dataDir, oldID, newID string) ([]Move, error) {
	for _, src := range config.SourceDirNames() {
		newDir := filepath.Join(dataDir, src, newID)
		if _, err := os.Stat(newDir); err == nil {
			return nil, fmt.Errorf("target data directory already exists: %s", newDir)
		} else if !os.IsNotExist(err) {
			return nil, err
		}
	}

	var moves []Move
	for _, src := range config.SourceDirNames() {
		oldDir := filepath.Join(dataDir, src, oldID)
		newDir := filepath.Join(dataDir, src, newID)
		if _, err := os.Stat(oldDir); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return moves, err
		}
		if err := os.MkdirAll(filepath.Dir(newDir), 0755); err != nil {
			return moves, err
		}
		if err := os.Rename(oldDir, newDir); err != nil {
			return moves, err
		}
		moves = append(moves, Move{from: oldDir, to: newDir, oldID: oldID, newID: newID})
		if err := store.RewriteProjectID(newDir, oldID, newID); err != nil {
			return moves, err
		}
	}
	return moves, nil
}

func RollbackMoves(moves []Move) error {
	for i := len(moves) - 1; i >= 0; i-- {
		move := moves[i]
		if _, err := os.Stat(move.to); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return err
		}
		if move.oldID != "" && move.newID != "" {
			if err := store.RewriteProjectID(move.to, move.newID, move.oldID); err != nil {
				return err
			}
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

func TrashProjectDirs(dataDir, id string) (string, []Move, error) {
	if _, err := os.Stat(dataDir); err != nil {
		if os.IsNotExist(err) {
			return "", nil, nil
		}
		return "", nil, err
	}

	trashRoot, err := os.MkdirTemp(dataDir, ".remove-"+id+"-")
	if err != nil {
		return "", nil, err
	}

	var moves []Move
	for _, src := range config.SourceDirNames() {
		oldDir := filepath.Join(dataDir, src, id)
		trashDir := filepath.Join(trashRoot, src)
		if _, err := os.Stat(oldDir); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			if rollbackErr := RollbackMoves(moves); rollbackErr != nil {
				return trashRoot, nil, fmt.Errorf("%w (rollback failed: %v; trash dir preserved: %s)", err, rollbackErr, trashRoot)
			}
			_ = os.RemoveAll(trashRoot)
			return "", nil, err
		}
		if err := os.MkdirAll(filepath.Dir(trashDir), 0755); err != nil {
			if rollbackErr := RollbackMoves(moves); rollbackErr != nil {
				return trashRoot, nil, fmt.Errorf("%w (rollback failed: %v; trash dir preserved: %s)", err, rollbackErr, trashRoot)
			}
			_ = os.RemoveAll(trashRoot)
			return "", nil, err
		}
		if err := os.Rename(oldDir, trashDir); err != nil {
			if rollbackErr := RollbackMoves(moves); rollbackErr != nil {
				return trashRoot, nil, fmt.Errorf("%w (rollback failed: %v; trash dir preserved: %s)", err, rollbackErr, trashRoot)
			}
			_ = os.RemoveAll(trashRoot)
			return "", nil, err
		}
		moves = append(moves, Move{from: oldDir, to: trashDir})
	}

	if len(moves) == 0 {
		_ = os.RemoveAll(trashRoot)
		return "", nil, nil
	}
	return trashRoot, moves, nil
}
