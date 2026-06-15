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
	var name, github, githubTraffic, githubEvents, pypi, cran, homebrew, plausible, openvsx string

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

			flagsProvided := github != "" || githubTraffic != "" || githubEvents != "" || pypi != "" || cran != "" ||
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
			if githubEvents != "" && !validGitHubRe.MatchString(githubEvents) {
				return fmt.Errorf("invalid GitHub repo %q: must be owner/repo", githubEvents)
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
				Name:          name,
				GitHub:        toStringList(github),
				GitHubTraffic: toStringList(githubTraffic),
				GitHubEvents:  toStringList(githubEvents),
				PyPI:          toStringList(pypi),
				CRAN:          toStringList(cran),
				Homebrew:      toStringList(homebrew),
				Plausible:     toStringList(plausible),
				OpenVSX:       toStringList(openvsx),
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
	cmd.Flags().StringVar(&githubTraffic, "github-traffic", "", "GitHub owner/repo for traffic data")
	cmd.Flags().StringVar(&githubEvents, "github-events", "", "GitHub owner/repo for event tracking")
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

	fmt.Fprintln(os.Stdout, "Tip: use commas to specify multiple values (e.g., owner/repo-a, owner/repo-b)")
	fmt.Fprintln(os.Stdout)

	name := prompt(os.Stdout, reader, "Name", id, "")
	github := prompt(os.Stdout, reader, "GitHub (owner/repo)", detected.GitHub, detected.GitHubSource)
	githubTraffic := prompt(os.Stdout, reader, "GitHub traffic (owner/repo)", detected.GitHub, detected.GitHubSource)
	githubEvents := prompt(os.Stdout, reader, "GitHub events (owner/repo)", detected.GitHub, detected.GitHubSource)
	pypi := prompt(os.Stdout, reader, "PyPI package", detected.PyPI, detected.PyPISource)
	cran := prompt(os.Stdout, reader, "CRAN package", detected.CRAN, detected.CRANSource)
	homebrew := prompt(os.Stdout, reader, "Homebrew formula", "", "")
	plausible := prompt(os.Stdout, reader, "Plausible site ID", "", "")
	openvsx := prompt(os.Stdout, reader, "OpenVSX extension", detected.OpenVSX, detected.OpenVSXSource)

	for _, repo := range parseCommaSeparated(github) {
		if !validGitHubRe.MatchString(repo) {
			return fmt.Errorf("invalid GitHub repo %q: must be owner/repo", repo)
		}
	}
	for _, repo := range parseCommaSeparated(githubTraffic) {
		if !validGitHubRe.MatchString(repo) {
			return fmt.Errorf("invalid GitHub repo %q: must be owner/repo", repo)
		}
	}
	for _, repo := range parseCommaSeparated(githubEvents) {
		if !validGitHubRe.MatchString(repo) {
			return fmt.Errorf("invalid GitHub repo %q: must be owner/repo", repo)
		}
	}

	proj := config.Project{
		Name:          name,
		GitHub:        toStringList(github),
		GitHubTraffic: toStringList(githubTraffic),
		GitHubEvents:  toStringList(githubEvents),
		PyPI:          toStringList(pypi),
		CRAN:          toStringList(cran),
		Homebrew:      toStringList(homebrew),
		Plausible:     toStringList(plausible),
		OpenVSX:       toStringList(openvsx),
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
	if !p.GitHubTraffic.IsEmpty() {
		sources = append(sources, "github-traffic")
	}
	if !p.GitHubEvents.IsEmpty() {
		sources = append(sources, "github-events")
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
	return parseCommaSeparated(s)
}

func parseCommaSeparated(s string) config.StringList {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	var result config.StringList
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
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
