package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"

	"github.com/posit-dev/velocirepo/internal/config"
	"github.com/spf13/cobra"
)

func initCmd() *cobra.Command {
	var dataDir string
	var dir string

	cmd := &cobra.Command{
		Use:     "init",
		Short:   "Create a new velocirepo.toml config file",
		GroupID: "project",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			if dir == "" {
				var err error
				dir, err = os.Getwd()
				if err != nil {
					return err
				}
			}

			cfgPath := filepath.Join(dir, "velocirepo.toml")
			if _, err := os.Stat(cfgPath); err == nil {
				return fmt.Errorf("velocirepo.toml already exists in %s (use 'add-project' to add projects)", dir)
			}

			if dataDir == "" {
				dataDir = "velocirepo/data"
			}

			content := fmt.Sprintf("[data]\ndir = %q\n", dataDir)

			if isInteractive() {
				detected := config.DetectAll(dir)
				reader := bufio.NewReader(os.Stdin)

				if detected.ProjectID != "" || detected.GitHub != "" {
					_, _ = fmt.Fprintln(out, "Detected project information:")
					if detected.GitHub != "" {
						_, _ = fmt.Fprintf(out, "  GitHub: %s (from %s)\n", detected.GitHub, detected.GitHubSource)
					}
					if detected.PyPI != "" {
						_, _ = fmt.Fprintf(out, "  PyPI: %s (from %s)\n", detected.PyPI, detected.PyPISource)
					}
					if detected.CRAN != "" {
						_, _ = fmt.Fprintf(out, "  CRAN: %s (from %s)\n", detected.CRAN, detected.CRANSource)
					}
					if detected.OpenVSX != "" {
						_, _ = fmt.Fprintf(out, "  OpenVSX: %s (from %s)\n", detected.OpenVSX, detected.OpenVSXSource)
					}
					_, _ = fmt.Fprintln(out)

					ok, err := confirm(out, reader, "Add this project to config?")
					if err == nil && ok {
						id, err := prompt(out, reader, "Project ID", detected.ProjectID, detected.IDSource)
						if err != nil {
							return err
						}
						proj := config.Project{
							Name:         id,
							GitHubEvents: toStringList(detected.GitHub),
							PyPI:         toStringList(detected.PyPI),
							CRAN:         toStringList(detected.CRAN),
							OpenVSX:      toStringList(detected.OpenVSX),
						}
						content += "\n" + formatSection(id, proj)
						_, _ = fmt.Fprintf(out, "\nAdded project '%s'\n", id)
					}
				}
			}

			if err := os.WriteFile(cfgPath, []byte(content), 0644); err != nil {
				return fmt.Errorf("write config: %w", err)
			}

			// Create data directory
			absDataDir := filepath.Join(dir, dataDir)
			if err := os.MkdirAll(absDataDir, 0755); err != nil {
				return fmt.Errorf("create data directory: %w", err)
			}

			_, _ = fmt.Fprintf(out, "Created %s\n", cfgPath)
			return nil
		},
	}

	cmd.Flags().StringVar(&dir, "dir", "", "directory to create config in (default: current directory)")
	cmd.Flags().StringVar(&dataDir, "data-dir", "velocirepo/data", "data directory path")

	return cmd
}

func formatSection(id string, p config.Project) string {
	s := fmt.Sprintf("[projects.%s]\nname = %q\n", id, p.Name)
	for _, src := range p.Sources() {
		if !src.Values.IsEmpty() {
			s += fmt.Sprintf("%s = %q\n", src.Name, src.Values.First())
		}
	}
	return s
}
