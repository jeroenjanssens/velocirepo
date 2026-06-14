package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jeroenjanssens/velocirepo/internal/config"
	"github.com/spf13/cobra"
)

func projectInitCmd() *cobra.Command {
	var dataDir string
	var dir string

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Create a new velocirepo.toml config file",
		RunE: func(cmd *cobra.Command, args []string) error {
			if dir == "" {
				var err error
				dir, err = os.Getwd()
				if err != nil {
					return err
				}
			}

			cfgPath := filepath.Join(dir, "velocirepo.toml")
			if _, err := os.Stat(cfgPath); err == nil {
				return fmt.Errorf("velocirepo.toml already exists in %s (use 'project add' to add projects)", dir)
			}

			if dataDir == "" {
				dataDir = "data"
			}

			content := fmt.Sprintf("[data]\ndir = %q\n", dataDir)

			if isInteractive() {
				detected := config.DetectAll(dir)
				reader := bufio.NewReader(os.Stdin)

				if detected.ProjectID != "" || detected.GitHub != "" {
					fmt.Fprintln(os.Stdout, "Detected project information:")
					if detected.GitHub != "" {
						fmt.Fprintf(os.Stdout, "  GitHub: %s (from %s)\n", detected.GitHub, detected.GitHubSource)
					}
					if detected.PyPI != "" {
						fmt.Fprintf(os.Stdout, "  PyPI: %s (from %s)\n", detected.PyPI, detected.PyPISource)
					}
					if detected.CRAN != "" {
						fmt.Fprintf(os.Stdout, "  CRAN: %s (from %s)\n", detected.CRAN, detected.CRANSource)
					}
					if detected.OpenVSX != "" {
						fmt.Fprintf(os.Stdout, "  OpenVSX: %s (from %s)\n", detected.OpenVSX, detected.OpenVSXSource)
					}
					fmt.Fprintln(os.Stdout)

					if confirm(os.Stdout, reader, "Add this project to config?") {
						id := prompt(os.Stdout, reader, "Project ID", detected.ProjectID, detected.IDSource)
						proj := config.Project{
							Name:   id,
							GitHub: detected.GitHub,
							PyPI:   detected.PyPI,
							CRAN:   detected.CRAN,
							OpenVSX: detected.OpenVSX,
						}
						content += "\n" + formatSection(id, proj)
						fmt.Fprintf(os.Stdout, "\nAdded project '%s'\n", id)
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

			fmt.Fprintf(os.Stdout, "Created %s\n", cfgPath)
			return nil
		},
	}

	cmd.Flags().StringVar(&dir, "dir", "", "directory to create config in (default: current directory)")
	cmd.Flags().StringVar(&dataDir, "data-dir", "data", "data directory path")

	return cmd
}

func formatSection(id string, p config.Project) string {
	s := fmt.Sprintf("[projects.%s]\nname = %q\n", id, p.Name)
	if p.GitHub != "" {
		s += fmt.Sprintf("github = %q\n", p.GitHub)
	}
	if p.PyPI != "" {
		s += fmt.Sprintf("pypi = %q\n", p.PyPI)
	}
	if p.CRAN != "" {
		s += fmt.Sprintf("cran = %q\n", p.CRAN)
	}
	if p.Homebrew != "" {
		s += fmt.Sprintf("homebrew = %q\n", p.Homebrew)
	}
	if p.Plausible != "" {
		s += fmt.Sprintf("plausible = %q\n", p.Plausible)
	}
	if p.OpenVSX != "" {
		s += fmt.Sprintf("openvsx = %q\n", p.OpenVSX)
	}
	return s
}
