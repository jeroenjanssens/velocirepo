package cmd

import (
	"fmt"

	"github.com/jeroenjanssens/velocirepo/internal/store"
	"github.com/jeroenjanssens/velocirepo/internal/ui"
	"github.com/spf13/cobra"
)

func migrateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "migrate",
		Short: "Migrate data to the latest schema version",
		RunE: func(cmd *cobra.Command, args []string) error {
			dataDir := cfg.DataDir()

			current, err := store.SchemaVersion(dataDir)
			if err != nil {
				return err
			}

			if current >= store.LatestSchemaVersion {
				ui.Infof("data is already at schema version %d (latest)", current)
				return nil
			}

			ui.Infof("migrating from schema version %d to %d", current, store.LatestSchemaVersion)
			for i := current + 1; i <= store.LatestSchemaVersion; i++ {
				ui.Infof("  %d: %s", i, store.MigrationDescription(i))
			}

			applied, err := store.Migrate(dataDir)
			if err != nil {
				return fmt.Errorf("migration failed after %d step(s): %w", applied, err)
			}

			ui.Infof("applied %d migration(s), now at schema version %d", applied, store.LatestSchemaVersion)
			return nil
		},
	}
}
