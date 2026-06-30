package cmd

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/jeroenjanssens/velocirepo/internal/config"
	"github.com/jeroenjanssens/velocirepo/internal/fetch"
	"github.com/spf13/cobra"
)

func importProjectsCmd() *cobra.Command {
	var (
		githubOrg       string
		githubUser      string
		fromFile        string
		filter          string
		includeForks    bool
		includeArchived bool
		dryRun          bool
		skipExisting    bool
		yes             bool
	)

	cmd := &cobra.Command{
		Use:     "import-projects",
		Short:   "Bulk-add projects from an external source",
		GroupID: "project",
		RunE: func(cmd *cobra.Command, args []string) error {
			sources := 0
			if githubOrg != "" {
				sources++
			}
			if githubUser != "" {
				sources++
			}
			if fromFile != "" {
				sources++
			}
			if sources == 0 {
				return fmt.Errorf("specify one of --github-org, --github-user, or --from-file")
			}
			if sources > 1 {
				return fmt.Errorf("specify only one of --github-org, --github-user, or --from-file")
			}

			var projects []fetch.ImportEntry

			importOpts := fetch.ImportOptions{
				IncludeForks:    includeForks,
				IncludeArchived: includeArchived,
				Filter:          filter,
			}

			switch {
			case githubOrg != "":
				var err error
				projects, err = fetch.FetchGitHubRepos(cmd.Context(), os.Getenv("GITHUB_TOKEN"), "orgs/"+githubOrg+"/repos", importOpts)
				if err != nil {
					return err
				}
			case githubUser != "":
				var err error
				projects, err = fetch.FetchGitHubRepos(cmd.Context(), os.Getenv("GITHUB_TOKEN"), "users/"+githubUser+"/repos", importOpts)
				if err != nil {
					return err
				}
			case fromFile != "":
				var err error
				projects, err = loadFromFile(fromFile)
				if err != nil {
					return err
				}
			}

			if len(projects) == 0 {
				fmt.Println("No projects found to import.")
				return nil
			}

			existing := cfg.Projects
			var toAdd []fetch.ImportEntry
			var skipped int
			for _, p := range projects {
				if _, exists := existing[p.ID]; exists {
					if skipExisting {
						skipped++
						continue
					}
					return fmt.Errorf("project %q already exists (use --skip-existing to skip)", p.ID)
				}
				toAdd = append(toAdd, p)
			}

			if len(toAdd) == 0 {
				fmt.Printf("No new projects to add (%d skipped as existing).\n", skipped)
				return nil
			}

			_, _ = fmt.Fprintf(os.Stdout, "Projects to add (%d):\n", len(toAdd))
			for _, p := range toAdd {
				_, _ = fmt.Fprintf(os.Stdout, "  %s (%s)\n", p.ID, p.Project.GitHubEvents.String())
			}
			if skipped > 0 {
				_, _ = fmt.Fprintf(os.Stdout, "  (%d skipped as existing)\n", skipped)
			}
			_, _ = fmt.Fprintln(os.Stdout)

			if dryRun {
				fmt.Println("Dry run — no changes made.")
				return nil
			}

			if !yes {
				if !isInteractive() {
					return fmt.Errorf("cannot prompt for confirmation (use --yes)")
				}
				reader := bufio.NewReader(os.Stdin)
				ok, err := confirm(os.Stdout, reader, "Add these projects?")
				if err != nil {
					return err
				}
				if !ok {
					fmt.Println("Cancelled.")
					return nil
				}
			}

			cfgPath := cfgFilePath()
			for _, p := range toAdd {
				if err := config.AppendProject(cfgPath, p.ID, p.Project); err != nil {
					return fmt.Errorf("add %s: %w", p.ID, err)
				}
			}

			_, _ = fmt.Fprintf(os.Stdout, "Added %d projects.\n", len(toAdd))
			rebuildDB()
			return nil
		},
	}

	cmd.Flags().StringVar(&githubOrg, "github-org", "", "import repos from a GitHub organization")
	cmd.Flags().StringVar(&githubUser, "github-user", "", "import repos from a GitHub user")
	cmd.Flags().StringVar(&fromFile, "from-file", "", "import from a CSV or JSON file")
	cmd.Flags().StringVar(&filter, "filter", "", "glob pattern to filter repo names")
	cmd.Flags().BoolVar(&includeForks, "include-forks", false, "include forked repos")
	cmd.Flags().BoolVar(&includeArchived, "include-archived", false, "include archived repos")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "show what would be added without writing")
	cmd.Flags().BoolVar(&skipExisting, "skip-existing", false, "skip projects that already exist")
	cmd.Flags().BoolVar(&yes, "yes", false, "skip confirmation prompt")

	return cmd
}

func loadFromFile(path string) ([]fetch.ImportEntry, error) {
	ext := strings.ToLower(filepath.Ext(path))

	switch ext {
	case ".json":
		return loadFromJSON(path)
	case ".csv":
		return loadFromCSV(path)
	default:
		return nil, fmt.Errorf("unsupported file format %q (use .json or .csv)", ext)
	}
}

func loadFromJSON(path string) ([]fetch.ImportEntry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var rawItems []map[string]json.RawMessage
	if err := json.Unmarshal(data, &rawItems); err != nil {
		return nil, fmt.Errorf("parse JSON: %w", err)
	}

	var entries []fetch.ImportEntry
	for _, item := range rawItems {
		id, err := jsonString(item, "id")
		if err != nil {
			return nil, err
		}
		if id == "" {
			return nil, fmt.Errorf("JSON entry missing 'id' field")
		}
		if !validIDRe.MatchString(id) {
			return nil, fmt.Errorf("invalid project ID %q: must be lowercase alphanumeric with hyphens", id)
		}
		name, err := jsonString(item, "name")
		if err != nil {
			return nil, err
		}
		if name == "" {
			name = id
		}
		project := config.Project{Name: name}
		if err := applyJSONSourceValues(&project, item); err != nil {
			return nil, err
		}
		entries = append(entries, fetch.ImportEntry{
			ID:      id,
			Project: project,
		})
	}

	return entries, nil
}

func loadFromCSV(path string) ([]fetch.ImportEntry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()

	reader := csv.NewReader(f)
	header, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("read CSV header: %w", err)
	}

	colIndex := make(map[string]int)
	for i, col := range header {
		colIndex[strings.TrimSpace(strings.ToLower(col))] = i
	}

	idIdx, hasID := colIndex["id"]
	if !hasID {
		return nil, fmt.Errorf("CSV missing 'id' column")
	}

	var entries []fetch.ImportEntry
	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read CSV row: %w", err)
		}

		id := row[idIdx]
		if id == "" {
			continue
		}
		if !validIDRe.MatchString(id) {
			return nil, fmt.Errorf("invalid project ID %q: must be lowercase alphanumeric with hyphens", id)
		}

		p := config.Project{}
		if idx, ok := colIndex["name"]; ok && idx < len(row) {
			p.Name = row[idx]
		}
		if p.Name == "" {
			p.Name = id
		}
		for _, field := range projectSourceFields {
			p.SetSourceValues(field.key, toStringList(sourceValueFromCSV(row, colIndex, field)))
		}

		entries = append(entries, fetch.ImportEntry{ID: id, Project: p})
	}

	return entries, nil
}
