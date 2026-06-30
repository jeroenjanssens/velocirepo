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

func addProjectCmd() *cobra.Command {
	var name string
	sourceInputs := newProjectSourceInputs()

	cmd := &cobra.Command{
		Use:     "add-project [id]",
		Short:   "Add a new project to the config",
		Args:    cobra.MaximumNArgs(1),
		GroupID: "project",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfgPath := cfgFilePath()

			var id string
			if len(args) > 0 {
				id = args[0]
			}

			flagsProvided := sourceInputs.anyNonEmpty()
			if !flagsProvided && isInteractive() {
				return projectAddInteractive(cfgPath, id)
			}

			if id == "" {
				return fmt.Errorf("project ID is required (or run interactively without flags)")
			}

			if !validIDRe.MatchString(id) {
				return fmt.Errorf("invalid project ID %q: must be lowercase alphanumeric with hyphens", id)
			}

			if _, err := cfg.GetProject(id); err == nil {
				return fmt.Errorf("project %q already exists (use 'update-project' to modify)", id)
			}

			if !flagsProvided {
				return fmt.Errorf("at least one source must be configured")
			}

			if name == "" {
				name = id
			}

			proj, err := sourceInputs.toProject(name)
			if err != nil {
				return err
			}

			if err := config.AppendProject(cfgPath, id, proj); err != nil {
				return err
			}

			sources := listSources(proj)
			_, _ = fmt.Fprintf(os.Stdout, "Added project '%s' with sources: %s\n", id, strings.Join(sources, ", "))
			rebuildDB()
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "display name")
	sourceInputs.registerFlags(cmd)

	return cmd
}

func projectAddInteractive(cfgPath string, id string) error {
	dir, _ := os.Getwd()
	detected := config.DetectAll(dir)
	reader := bufio.NewReader(os.Stdin)

	suppress := false
	if id == "" {
		var overridden bool
		var err error
		id, overridden, err = promptWithHint(os.Stdout, reader, "Project ID", detected.ProjectID, detected.IDSource, suppress)
		if err != nil {
			return err
		}
		if overridden {
			suppress = true
		}
	}
	if id == "" {
		return fmt.Errorf("project ID is required")
	}
	if !validIDRe.MatchString(id) {
		return fmt.Errorf("invalid project ID %q: must be lowercase alphanumeric with hyphens", id)
	}

	if _, err := cfg.GetProject(id); err == nil {
		return fmt.Errorf("project %q already exists (use 'update-project' to modify)", id)
	}

	_, _ = fmt.Fprintln(os.Stdout, "Tip: use commas to specify multiple values (e.g., owner/repo-a, owner/repo-b)")
	_, _ = fmt.Fprintln(os.Stdout)

	var overridden bool

	name, overridden, err := promptWithHint(os.Stdout, reader, "Name", id, "", suppress)
	if err != nil {
		return err
	}
	if overridden {
		suppress = true
	}

	values := make(map[string]string, len(projectSourceFields))
	for _, field := range projectSourceFields {
		hint, hintSource := "", ""
		if field.detected != nil {
			hint, hintSource = field.detected(detected)
		}
		value, overridden, err := promptWithHint(os.Stdout, reader, field.addPrompt, hint, hintSource, suppress)
		if err != nil {
			return err
		}
		if overridden {
			suppress = true
		}
		values[field.key] = value
	}

	proj, err := projectFromRawSourceValues(name, values)
	if err != nil {
		return err
	}

	sources := listSources(proj)
	if len(sources) == 0 {
		return fmt.Errorf("at least one source must be configured")
	}

	if err := config.AppendProject(cfgPath, id, proj); err != nil {
		return err
	}

	_, _ = fmt.Fprintf(os.Stdout, "\nAdded project '%s' with sources: %s\n", id, strings.Join(sources, ", "))
	rebuildDB()
	return nil
}

func listSources(p config.Project) []string {
	return p.SourceNames()
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
