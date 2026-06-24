package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jeroenjanssens/velocirepo/internal/config"
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

					ok, err := confirm(os.Stdout, reader, "Add this project to config?")
					if err == nil && ok {
						id, err := prompt(os.Stdout, reader, "Project ID", detected.ProjectID, detected.IDSource)
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
	cmd.Flags().StringVar(&dataDir, "data-dir", "velocirepo/data", "data directory path")

	return cmd
}

func formatSection(id string, p config.Project) string {
	s := fmt.Sprintf("[projects.%s]\nname = %q\n", id, p.Name)
	if !p.GitHubEvents.IsEmpty() {
		s += fmt.Sprintf("github = %q\n", p.GitHubEvents.First())
	}
	if !p.GitHubTraffic.IsEmpty() {
		s += fmt.Sprintf("github-traffic = %q\n", p.GitHubTraffic.First())
	}
	if !p.PyPI.IsEmpty() {
		s += fmt.Sprintf("pypi = %q\n", p.PyPI.First())
	}
	if !p.CRAN.IsEmpty() {
		s += fmt.Sprintf("cran = %q\n", p.CRAN.First())
	}
	if !p.Homebrew.IsEmpty() {
		s += fmt.Sprintf("homebrew = %q\n", p.Homebrew.First())
	}
	if !p.Plausible.IsEmpty() {
		s += fmt.Sprintf("plausible = %q\n", p.Plausible.First())
	}
	if !p.OpenVSX.IsEmpty() {
		s += fmt.Sprintf("openvsx = %q\n", p.OpenVSX.First())
	}
	return s
}
