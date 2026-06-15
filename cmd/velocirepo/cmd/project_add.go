package cmd

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/jeroenjanssens/velocirepo/internal/config"
	"github.com/spf13/cobra"
)

var validIDRe = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)
var validGitHubRe = regexp.MustCompile(`^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$`)

func projectAddCmd() *cobra.Command {
	var name, github, pypi, cran, homebrew, plausible, openvsx string

	cmd := &cobra.Command{
		Use:   "add [id]",
		Short: "Add a new project to the config",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfgPath := cfgFilePath()

			var id string
			if len(args) > 0 {
				id = args[0]
			}

			flagsProvided := github != "" || pypi != "" || cran != "" ||
				homebrew != "" || plausible != "" || openvsx != ""

			if !flagsProvided && isInteractive() {
				return projectAddInteractive(cfgPath, id)
			}

			if id == "" {
				return fmt.Errorf("project ID is required (or run interactively without flags)")
			}

			if !validIDRe.MatchString(id) {
				return fmt.Errorf("invalid project ID %q: must be lowercase alphanumeric with hyphens", id)
			}
			if github != "" && !validGitHubRe.MatchString(github) {
				return fmt.Errorf("invalid GitHub repo %q: must be owner/repo", github)
			}

			projects := cfg.ResolveProjects()
			if _, exists := projects[id]; exists {
				return fmt.Errorf("project %q already exists (use 'project update' to modify)", id)
			}

			if !flagsProvided {
				return fmt.Errorf("at least one source must be configured")
			}

			if name == "" {
				name = id
			}

			proj := config.Project{
				Name:      name,
				GitHub:    toStringList(github),
				PyPI:      toStringList(pypi),
				CRAN:      toStringList(cran),
				Homebrew:  toStringList(homebrew),
				Plausible: toStringList(plausible),
				OpenVSX:   toStringList(openvsx),
			}

			if err := config.AppendProject(cfgPath, id, proj); err != nil {
				return err
			}

			sources := listSources(proj)
			fmt.Fprintf(os.Stdout, "Added project '%s' with sources: %s\n", id, strings.Join(sources, ", "))
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "display name")
	cmd.Flags().StringVar(&github, "github", "", "GitHub owner/repo")
	cmd.Flags().StringVar(&pypi, "pypi", "", "PyPI package name")
	cmd.Flags().StringVar(&cran, "cran", "", "CRAN package name")
	cmd.Flags().StringVar(&homebrew, "homebrew", "", "Homebrew formula (user/tap/formula)")
	cmd.Flags().StringVar(&plausible, "plausible", "", "Plausible site ID")
	cmd.Flags().StringVar(&openvsx, "openvsx", "", "OpenVSX extension (publisher/extension)")

	return cmd
}

func projectAddInteractive(cfgPath string, id string) error {
	dir, _ := os.Getwd()
	detected := config.DetectAll(dir)
	reader := bufio.NewReader(os.Stdin)

	if id == "" {
		id = prompt(os.Stdout, reader, "Project ID", detected.ProjectID, detected.IDSource)
	}
	if id == "" {
		return fmt.Errorf("project ID is required")
	}
	if !validIDRe.MatchString(id) {
		return fmt.Errorf("invalid project ID %q: must be lowercase alphanumeric with hyphens", id)
	}

	projects := cfg.ResolveProjects()
	if _, exists := projects[id]; exists {
		return fmt.Errorf("project %q already exists (use 'project update' to modify)", id)
	}

	name := prompt(os.Stdout, reader, "Name", id, "")
	github := prompt(os.Stdout, reader, "GitHub (owner/repo)", detected.GitHub, detected.GitHubSource)
	pypi := prompt(os.Stdout, reader, "PyPI package", detected.PyPI, detected.PyPISource)
	cran := prompt(os.Stdout, reader, "CRAN package", detected.CRAN, detected.CRANSource)
	homebrew := prompt(os.Stdout, reader, "Homebrew formula", "", "")
	plausible := prompt(os.Stdout, reader, "Plausible site ID", "", "")
	openvsx := prompt(os.Stdout, reader, "OpenVSX extension", detected.OpenVSX, detected.OpenVSXSource)

	if github != "" && !validGitHubRe.MatchString(github) {
		return fmt.Errorf("invalid GitHub repo %q: must be owner/repo", github)
	}

	proj := config.Project{
		Name:      name,
		GitHub:    toStringList(github),
		PyPI:      toStringList(pypi),
		CRAN:      toStringList(cran),
		Homebrew:  toStringList(homebrew),
		Plausible: toStringList(plausible),
		OpenVSX:   toStringList(openvsx),
	}

	sources := listSources(proj)
	if len(sources) == 0 {
		return fmt.Errorf("at least one source must be configured")
	}

	if err := config.AppendProject(cfgPath, id, proj); err != nil {
		return err
	}

	fmt.Fprintf(os.Stdout, "\nAdded project '%s' with sources: %s\n", id, strings.Join(sources, ", "))
	return nil
}

func listSources(p config.Project) []string {
	var sources []string
	if !p.GitHub.IsEmpty() {
		sources = append(sources, "github")
	}
	if !p.PyPI.IsEmpty() {
		sources = append(sources, "pypi")
	}
	if !p.CRAN.IsEmpty() {
		sources = append(sources, "cran")
	}
	if !p.Homebrew.IsEmpty() {
		sources = append(sources, "homebrew")
	}
	if !p.Plausible.IsEmpty() {
		sources = append(sources, "plausible")
	}
	if !p.OpenVSX.IsEmpty() {
		sources = append(sources, "openvsx")
	}
	return sources
}

func toStringList(s string) config.StringList {
	if s == "" {
		return nil
	}
	return config.StringList{s}
}

func cfgFilePath() string {
	if cfgFile != "" {
		return cfgFile
	}
	if env := os.Getenv("VELOCIREPO_CONFIG"); env != "" {
		return env
	}
	if cfg != nil && cfg.Dir != "" {
		return cfg.Dir + "/velocirepo.toml"
	}
	return "velocirepo.toml"
}
