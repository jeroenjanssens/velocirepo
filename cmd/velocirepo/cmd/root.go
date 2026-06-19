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
		Short: "Fetch and aggregate open-source project metrics",
		Long:  "velocirepo collects metrics from GitHub, PyPI, CRAN, Plausible, and OpenVSX for your open-source projects.",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true
			setupLogging()

			if cmd.Name() == "version" || cmd.CommandPath() == "velocirepo project init" {
				godotenv.Load()
				return nil
			}

			var err error
			cfg, err = config.Load(cfgFile)
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}

			godotenv.Load(filepath.Join(cfg.Dir, ".env"))

			if cmd.Name() != "migrate" {
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

	rootCmd.AddCommand(versionCmd())
	rootCmd.AddCommand(fetchCmd())
	rootCmd.AddCommand(queryCmd())
	rootCmd.AddCommand(exportCmd())
	rootCmd.AddCommand(badgeCmd())
	rootCmd.AddCommand(ciCmd())
	rootCmd.AddCommand(projectCmd())
	rootCmd.AddCommand(migrateCmd())

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
