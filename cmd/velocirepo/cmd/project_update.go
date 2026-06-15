package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/jeroenjanssens/velocirepo/internal/config"
	"github.com/spf13/cobra"
)

func projectUpdateCmd() *cobra.Command {
	var name, github, pypi, cran, homebrew, plausible, openvsx string
	var unset []string

	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a project's configuration",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]

			projects := cfg.ResolveProjects()
			proj, exists := projects[id]
			if !exists {
				return fmt.Errorf("project %q not found in config (use 'project add' to create)", id)
			}

			cfgPath := cfgFilePath()

			flagsProvided := cmd.Flags().Changed("name") || cmd.Flags().Changed("github") ||
				cmd.Flags().Changed("pypi") || cmd.Flags().Changed("cran") ||
				cmd.Flags().Changed("homebrew") || cmd.Flags().Changed("plausible") ||
				cmd.Flags().Changed("openvsx") || len(unset) > 0

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
				if github != "" && !validGitHubRe.MatchString(github) {
					return fmt.Errorf("invalid GitHub repo %q: must be owner/repo", github)
				}
				updates["github"] = github
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
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "display name")
	cmd.Flags().StringVar(&github, "github", "", "GitHub owner/repo")
	cmd.Flags().StringVar(&pypi, "pypi", "", "PyPI package name")
	cmd.Flags().StringVar(&cran, "cran", "", "CRAN package name")
	cmd.Flags().StringVar(&homebrew, "homebrew", "", "Homebrew formula")
	cmd.Flags().StringVar(&plausible, "plausible", "", "Plausible site ID")
	cmd.Flags().StringVar(&openvsx, "openvsx", "", "OpenVSX extension")
	cmd.Flags().StringSliceVar(&unset, "unset", nil, "fields to remove (can be repeated)")

	return cmd
}

func projectUpdateInteractive(cfgPath string, id string, proj config.Project) error {
	reader := bufio.NewReader(os.Stdin)
	fmt.Fprintf(os.Stdout, "Updating project '%s' (press Enter to keep current value):\n\n", id)

	name := prompt(os.Stdout, reader, "Name", proj.Name, "")
	github := prompt(os.Stdout, reader, "GitHub (owner/repo)", proj.GitHub.First(), "")
	pypi := prompt(os.Stdout, reader, "PyPI package", proj.PyPI.First(), "")
	cran := prompt(os.Stdout, reader, "CRAN package", proj.CRAN.First(), "")
	homebrew := prompt(os.Stdout, reader, "Homebrew formula", proj.Homebrew.First(), "")
	plausible := prompt(os.Stdout, reader, "Plausible site ID", proj.Plausible.First(), "")
	openvsx := prompt(os.Stdout, reader, "OpenVSX extension", proj.OpenVSX.First(), "")

	if github != "" && !validGitHubRe.MatchString(github) {
		return fmt.Errorf("invalid GitHub repo %q: must be owner/repo", github)
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
	updateOrUnset("github", github, proj.GitHub.First())
	updateOrUnset("pypi", pypi, proj.PyPI.First())
	updateOrUnset("cran", cran, proj.CRAN.First())
	updateOrUnset("homebrew", homebrew, proj.Homebrew.First())
	updateOrUnset("plausible", plausible, proj.Plausible.First())
	updateOrUnset("openvsx", openvsx, proj.OpenVSX.First())

	if len(updates) == 0 && len(unsets) == 0 {
		fmt.Println("No changes.")
		return nil
	}

	if err := config.UpdateProject(cfgPath, id, updates, unsets); err != nil {
		return err
	}

	fmt.Fprintf(os.Stdout, "\nUpdated project '%s'\n", id)
	return nil
}
