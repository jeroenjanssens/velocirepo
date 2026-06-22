package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/jeroenjanssens/velocirepo/internal/config"
	"github.com/jeroenjanssens/velocirepo/internal/store"
	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
)

var (
	cfgFile string
	verbose bool
	quiet   bool
	cfg     *config.Config
)

func newRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "velocirepo",
		Short: "Fetch and store open-source project metrics",
		Long:  "velocirepo collects metrics from GitHub, PyPI, CRAN, Plausible, and OpenVSX for your open-source projects.",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true
			setupLogging()

			if cmd.Name() == "version" || cmd.Name() == "init" {
				return nil
			}

			var err error
			cfg, err = config.Load(cfgFile)
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}

			godotenv.Load(filepath.Join(cfg.Dir, ".env"))

			if cmd.Name() != "migrate" && cmd.Name() != "mcp" {
				if err := store.CheckSchemaVersion(cfg.DataDir()); err != nil {
					return err
				}
			}

			return nil
		},
	}

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default: walk up for velocirepo.toml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable debug logging")
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "suppress info messages")

	rootCmd.AddGroup(
		&cobra.Group{ID: "fetch", Title: "Fetching:"},
		&cobra.Group{ID: "query", Title: "Querying:"},
		&cobra.Group{ID: "badge", Title: "Badges:"},
		&cobra.Group{ID: "view", Title: "Views:"},
		&cobra.Group{ID: "project", Title: "Projects:"},
		&cobra.Group{ID: "ci", Title: "CI/CD:"},
		&cobra.Group{ID: "data", Title: "Data:"},
	)

	// Fetching
	rootCmd.AddCommand(fetchAllCmd())
	rootCmd.AddCommand(fetchGitHubCmd())
	rootCmd.AddCommand(fetchGitHubTrafficCmd())
	rootCmd.AddCommand(fetchPyPICmd())
	rootCmd.AddCommand(fetchCRANCmd())
	rootCmd.AddCommand(fetchHomebrewCmd())
	rootCmd.AddCommand(fetchPlausibleCmd())
	rootCmd.AddCommand(fetchOpenVSXCmd())
	rootCmd.AddCommand(fetchYouTubeCmd())

	// Querying
	rootCmd.AddCommand(queryCmd())
	rootCmd.AddCommand(schemaCmd())
	rootCmd.AddCommand(exportCmd())

	// Badges
	rootCmd.AddCommand(badgeCmd())

	// Projects
	rootCmd.AddCommand(initCmd())
	rootCmd.AddCommand(addProjectCmd())
	rootCmd.AddCommand(removeProjectCmd())
	rootCmd.AddCommand(renameProjectCmd())
	rootCmd.AddCommand(updateProjectCmd())
	rootCmd.AddCommand(showProjectCmd())
	rootCmd.AddCommand(listProjectsCmd())
	rootCmd.AddCommand(importProjectsCmd())
	rootCmd.AddCommand(validateProjectsCmd())

	// Views
	rootCmd.AddCommand(addViewCmd())
	rootCmd.AddCommand(removeViewCmd())
	rootCmd.AddCommand(listViewsCmd())
	rootCmd.AddCommand(showViewCmd())
	rootCmd.AddCommand(renderViewCmd())
	rootCmd.AddCommand(renderViewsCmd())
	rootCmd.AddCommand(serveViewCmd())

	// CI/CD
	rootCmd.AddCommand(installCICmd())
	rootCmd.AddCommand(syncSecretsCmd())

	// Data
	rootCmd.AddCommand(migrateCmd())

	// MCP
	rootCmd.AddCommand(mcpCmd())

	// Other
	rootCmd.AddCommand(versionCmd())

	return rootCmd
}

func setupLogging() {
	level := slog.LevelError + 1 // suppress all by default
	if verbose {
		level = slog.LevelDebug
	}

	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level})
	slog.SetDefault(slog.New(handler))
}

func Execute(ctx context.Context) error {
	return newRootCmd().ExecuteContext(ctx)
}
