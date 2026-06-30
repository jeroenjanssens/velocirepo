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
	var name string
	var unset []string
	sourceInputs := newProjectSourceInputs()

	cmd := &cobra.Command{
		Use:     "update-project <id>",
		Short:   "Update a project's configuration",
		Args:    cobra.ExactArgs(1),
		GroupID: "project",
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]

			proj, err := cfg.GetProject(id)
			if err != nil {
				return fmt.Errorf("%w (use 'add-project' to create)", err)
			}

			cfgPath := cfgFilePath()

			flagsProvided := cmd.Flags().Changed("name") ||
				sourceFlagsChanged(cmd) || len(unset) > 0

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
			sourceUpdates, err := sourceInputs.changedUpdates(cmd)
			if err != nil {
				return err
			}
			for key, value := range sourceUpdates {
				updates[key] = value
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
			_, _ = fmt.Fprintf(os.Stdout, "Updated project '%s': %s\n", id, strings.Join(changes, ", "))
			rebuildDB()
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "display name")
	sourceInputs.registerFlags(cmd)
	cmd.Flags().StringSliceVar(&unset, "unset", nil, "fields to remove (can be repeated)")

	return cmd
}

func sourceFlagsChanged(cmd *cobra.Command) bool {
	for _, field := range projectSourceFields {
		if cmd.Flags().Changed(field.CLIFlag) {
			return true
		}
	}
	return false
}

func projectUpdateInteractive(cfgPath string, id string, proj config.Project) error {
	reader := bufio.NewReader(os.Stdin)
	_, _ = fmt.Fprintf(os.Stdout, "Updating project '%s' (press Enter to keep current value):\n", id)
	_, _ = fmt.Fprintln(os.Stdout, "Tip: use commas to specify multiple values (e.g., owner/repo-a, owner/repo-b)")
	_, _ = fmt.Fprintln(os.Stdout)

	name, err := prompt(os.Stdout, reader, "Name", proj.Name, "")
	if err != nil {
		return err
	}

	sourceValues := make(map[string]string, len(projectSourceFields))
	for _, field := range projectSourceFields {
		value, err := prompt(os.Stdout, reader, field.UpdatePrompt, proj.SourceValues(field.Name).String(), "")
		if err != nil {
			return err
		}
		if err := validateProjectSourceValue(field, value); err != nil {
			return err
		}
		sourceValues[field.Name] = value
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
	for _, field := range projectSourceFields {
		updateOrUnset(field.TOMLKey, sourceValues[field.Name], proj.SourceValues(field.Name).String())
	}

	if len(updates) == 0 && len(unsets) == 0 {
		fmt.Println("No changes.")
		return nil
	}

	if err := config.UpdateProject(cfgPath, id, updates, unsets); err != nil {
		return err
	}

	_, _ = fmt.Fprintf(os.Stdout, "\nUpdated project '%s'\n", id)
	rebuildDB()
	return nil
}
