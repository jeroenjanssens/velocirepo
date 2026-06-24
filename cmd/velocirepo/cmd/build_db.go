package cmd

import (
	"time"

	"github.com/jeroenjanssens/velocirepo/internal/config"
	"github.com/jeroenjanssens/velocirepo/internal/store"
	"github.com/jeroenjanssens/velocirepo/internal/ui"
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
			if err := store.BuildDB(dataDir, projectInfos()); err != nil {
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
	projects := fresh.ResolveProjects()
	var infos []store.ProjectInfo
	for id, p := range projects {
		infos = append(infos, store.ProjectInfo{
			ID:          id,
			Name:        p.Name,
			Description: p.Description,
			Color:       p.Color,
			Tags:        []string(p.Tags),
			Website:     p.Website,
			Logo:        p.Logo,
		})
	}
	if err := store.BuildDB(cfg.DataDir(), infos); err != nil {
		ui.Warnf("build-db: %v", err)
	} else {
		ui.Infof("Updated velocirepo.duckdb")
	}
}
