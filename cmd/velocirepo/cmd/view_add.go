package cmd

import (
	"fmt"
	"os"

	"github.com/jeroenjanssens/velocirepo/internal/views"
	"github.com/spf13/cobra"
)

func addViewCmd() *cobra.Command {
	var framework, source, venv, output string

	cmd := &cobra.Command{
		Use:     "add-view <name>",
		Short:   "Scaffold a new view",
		Long:    "Create a new view file from a template. The name can include slashes for subdirectories (e.g., weekly/stars).",
		Args:    cobra.ExactArgs(1),
		GroupID: "view",
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			fw, err := views.ParseFramework(framework)
			if err != nil {
				return err
			}

			if source == "" {
				source = cfg.ViewsSource()
			}
			if source != "parquet" && source != "jsonl" {
				return fmt.Errorf("invalid source %q (use parquet or jsonl)", source)
			}

			ver, err := views.CheckRenderer(fw, venv)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
			} else if ver != "" && !quiet {
				fmt.Fprintf(os.Stderr, "Using %s %s\n", fw, ver)
			}

			viewsDir := cfg.ViewsDir()
			path, err := views.Scaffold(viewsDir, name, fw, source, cfg.DataDir())
			if err != nil {
				return err
			}

			fmt.Printf("Created view '%s' at %s\n", name, path)

			if venv != "" || output != "" || source != cfg.ViewsSource() {
				fmt.Fprintf(os.Stderr, "Note: add a [[views.items]] entry to velocirepo.toml for custom options (venv, output, source)\n")
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&framework, "framework", "f", "", "framework: quarto, jupyter, marimo, r, sql (required)")
	cmd.Flags().StringVarP(&source, "source", "s", "", "data source: parquet or jsonl (default: from config)")
	cmd.Flags().StringVar(&venv, "venv", "", "Python venv path")
	cmd.Flags().StringVarP(&output, "output", "o", "", "custom output path")
	cmd.MarkFlagRequired("framework")

	return cmd
}
