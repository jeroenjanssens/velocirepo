package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/jeroenjanssens/velocirepo/internal/config"
	"github.com/spf13/cobra"
)

func updateProjectCmd() *cobra.Command {
	var name, githubEvents, githubTraffic, pypi, cran, homebrew, plausible, openvsx, youtube string
	var unset []string

	cmd := &cobra.Command{
		Use:     "update-project <id>",
		Short:   "Update a project's configuration",
		Args:    cobra.ExactArgs(1),
		GroupID: "project",
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]

			projects := cfg.ResolveProjects()
			proj, exists := projects[id]
			if !exists {
				return fmt.Errorf("project %q not found in config (use 'add-project' to create)", id)
			}

			cfgPath := cfgFilePath()

			flagsProvided := cmd.Flags().Changed("name") ||
				cmd.Flags().Changed("github") || cmd.Flags().Changed("github-traffic") ||
				cmd.Flags().Changed("pypi") || cmd.Flags().Changed("cran") || cmd.Flags().Changed("homebrew") ||
				cmd.Flags().Changed("plausible") || cmd.Flags().Changed("openvsx") ||
				cmd.Flags().Changed("youtube") || len(unset) > 0

			if !flagsProvided && isInteractive() {
				return projectUpdateInteractive(cfgPath, id, proj)
			}

			if !flagsProvided {
				return fmt.Errorf("no changes specified (use flags or run interactively)")
			}

			updates := make(map[string]string)
			if cmd.Flags().Changed("name") {
				updates["name"] = name
			}
			if cmd.Flags().Changed("github") {
				if githubEvents != "" && !validGitHubRe.MatchString(githubEvents) {
					return fmt.Errorf("invalid GitHub repo %q: must be owner/repo", githubEvents)
				}
				updates["github"] = githubEvents
			}
			if cmd.Flags().Changed("github-traffic") {
				if githubTraffic != "" && !validGitHubRe.MatchString(githubTraffic) {
					return fmt.Errorf("invalid GitHub repo %q: must be owner/repo", githubTraffic)
				}
				updates["github-traffic"] = githubTraffic
			}
			if cmd.Flags().Changed("pypi") {
				updates["pypi"] = pypi
			}
			if cmd.Flags().Changed("cran") {
				updates["cran"] = cran
			}
			if cmd.Flags().Changed("homebrew") {
				updates["homebrew"] = homebrew
			}
			if cmd.Flags().Changed("plausible") {
				updates["plausible"] = plausible
			}
			if cmd.Flags().Changed("openvsx") {
				updates["openvsx"] = openvsx
			}
			if cmd.Flags().Changed("youtube") {
				updates["youtube"] = youtube
			}

			if err := config.UpdateProject(cfgPath, id, updates, unset); err != nil {
				return err
			}

			var changes []string
			for k, v := range updates {
				changes = append(changes, fmt.Sprintf("%s=%q", k, v))
			}
			for _, u := range unset {
				changes = append(changes, fmt.Sprintf("-%s", u))
			}
			fmt.Fprintf(os.Stdout, "Updated project '%s': %s\n", id, strings.Join(changes, ", "))
			rebuildDB()
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "display name")
	cmd.Flags().StringVar(&githubEvents, "github", "", "GitHub owner/repo")
	cmd.Flags().StringVar(&githubTraffic, "github-traffic", "", "GitHub owner/repo for traffic data")
	cmd.Flags().StringVar(&pypi, "pypi", "", "PyPI package name")
	cmd.Flags().StringVar(&cran, "cran", "", "CRAN package name")
	cmd.Flags().StringVar(&homebrew, "homebrew", "", "Homebrew formula")
	cmd.Flags().StringVar(&plausible, "plausible", "", "Plausible site ID")
	cmd.Flags().StringVar(&openvsx, "openvsx", "", "OpenVSX extension")
	cmd.Flags().StringVar(&youtube, "youtube", "", "YouTube channel, playlist, or video ID")
	cmd.Flags().StringSliceVar(&unset, "unset", nil, "fields to remove (can be repeated)")

	return cmd
}

func projectUpdateInteractive(cfgPath string, id string, proj config.Project) error {
	reader := bufio.NewReader(os.Stdin)
	fmt.Fprintf(os.Stdout, "Updating project '%s' (press Enter to keep current value):\n", id)
	fmt.Fprintln(os.Stdout, "Tip: use commas to specify multiple values (e.g., owner/repo-a, owner/repo-b)")
	fmt.Fprintln(os.Stdout)

	name, err := prompt(os.Stdout, reader, "Name", proj.Name, "")
	if err != nil {
		return err
	}
	githubEvents, err := prompt(os.Stdout, reader, "GitHub (owner/repo)", proj.GitHubEvents.String(), "")
	if err != nil {
		return err
	}
	githubTraffic, err := prompt(os.Stdout, reader, "GitHub traffic (owner/repo)", proj.GitHubTraffic.String(), "")
	if err != nil {
		return err
	}
	pypi, err := prompt(os.Stdout, reader, "PyPI package", proj.PyPI.String(), "")
	if err != nil {
		return err
	}
	cran, err := prompt(os.Stdout, reader, "CRAN package", proj.CRAN.String(), "")
	if err != nil {
		return err
	}
	homebrew, err := prompt(os.Stdout, reader, "Homebrew formula", proj.Homebrew.String(), "")
	if err != nil {
		return err
	}
	plausible, err := prompt(os.Stdout, reader, "Plausible site ID", proj.Plausible.String(), "")
	if err != nil {
		return err
	}
	openvsx, err := prompt(os.Stdout, reader, "OpenVSX extension", proj.OpenVSX.String(), "")
	if err != nil {
		return err
	}
	youtube, err := prompt(os.Stdout, reader, "YouTube (@handle, PLxxx, or video ID)", proj.YouTube.String(), "")
	if err != nil {
		return err
	}

	for _, repo := range parseCommaSeparated(githubEvents) {
		if !validGitHubRe.MatchString(repo) {
			return fmt.Errorf("invalid GitHub repo %q: must be owner/repo", repo)
		}
	}
	for _, repo := range parseCommaSeparated(githubTraffic) {
		if !validGitHubRe.MatchString(repo) {
			return fmt.Errorf("invalid GitHub repo %q: must be owner/repo", repo)
		}
	}

	updates := make(map[string]string)
	var unsets []string

	updateOrUnset := func(key, newVal, oldVal string) {
		if newVal != oldVal {
			if newVal == "" {
				unsets = append(unsets, key)
			} else {
				updates[key] = newVal
			}
		}
	}

	updateOrUnset("name", name, proj.Name)
	updateOrUnset("github", githubEvents, proj.GitHubEvents.String())
	updateOrUnset("github-traffic", githubTraffic, proj.GitHubTraffic.String())
	updateOrUnset("pypi", pypi, proj.PyPI.String())
	updateOrUnset("cran", cran, proj.CRAN.String())
	updateOrUnset("homebrew", homebrew, proj.Homebrew.String())
	updateOrUnset("plausible", plausible, proj.Plausible.String())
	updateOrUnset("openvsx", openvsx, proj.OpenVSX.String())
	updateOrUnset("youtube", youtube, proj.YouTube.String())

	if len(updates) == 0 && len(unsets) == 0 {
		fmt.Println("No changes.")
		return nil
	}

	if err := config.UpdateProject(cfgPath, id, updates, unsets); err != nil {
		return err
	}

	fmt.Fprintf(os.Stdout, "\nUpdated project '%s'\n", id)
	rebuildDB()
	return nil
}
