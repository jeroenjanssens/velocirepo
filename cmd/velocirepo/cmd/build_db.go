package cmd

import (
	"time"

	"github.com/posit-dev/velocirepo/internal/config"
	"github.com/posit-dev/velocirepo/internal/store"
	"github.com/posit-dev/velocirepo/internal/ui"
	"github.com/spf13/cobra"
)

func buildDBCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "build-db",
		Short:   "Build the DuckDB database file for external tools",
		GroupID: "data",
		RunE: func(cmd *cobra.Command, args []string) error {
			started := time.Now()
			dataDir := cfg.DataDir()
			if err := store.BuildDB(dataDir, projectInfos(), indicatorDefs()); err != nil {
				return err
			}
			ui.Infof("Built %s/velocirepo.duckdb in %s", dataDir, time.Since(started).Round(time.Millisecond))
			return nil
		},
	}
	return cmd
}

func rebuildDB() {
	fresh, err := config.Load(cfgFilePath())
	if err != nil {
		ui.Warnf("build-db: reload config: %v", err)
		return
	}
	if err := store.BuildDB(cfg.DataDir(), fresh.ProjectInfos(), fresh.IndicatorDefs()); err != nil {
		ui.Warnf("build-db: %v", err)
	} else {
		ui.Infof("Updated velocirepo.duckdb")
	}
}
