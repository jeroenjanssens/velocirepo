package cmd

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jeroenjanssens/velocirepo/internal/config"
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

			var projects []importEntry

			switch {
			case githubOrg != "":
				var err error
				projects, err = fetchGitHubRepos(cmd, "orgs/"+githubOrg+"/repos", filter, includeForks, includeArchived)
				if err != nil {
					return err
				}
			case githubUser != "":
				var err error
				projects, err = fetchGitHubRepos(cmd, "users/"+githubUser+"/repos", filter, includeForks, includeArchived)
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

			existing := cfg.ResolveProjects()
			var toAdd []importEntry
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

			fmt.Fprintf(os.Stdout, "Projects to add (%d):\n", len(toAdd))
			for _, p := range toAdd {
				fmt.Fprintf(os.Stdout, "  %s (%s)\n", p.ID, p.Project.GitHubEvents.String())
			}
			if skipped > 0 {
				fmt.Fprintf(os.Stdout, "  (%d skipped as existing)\n", skipped)
			}
			fmt.Fprintln(os.Stdout)

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

			fmt.Fprintf(os.Stdout, "Added %d projects.\n", len(toAdd))
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

type importEntry struct {
	ID      string
	Project config.Project
}

func fetchGitHubRepos(cmd *cobra.Command, endpoint string, filter string, includeForks, includeArchived bool) ([]importEntry, error) {
	token := os.Getenv("GITHUB_TOKEN")
	client := &http.Client{Timeout: 30 * time.Second}

	var allRepos []importEntry
	page := 1

	for {
		url := fmt.Sprintf("https://api.github.com/%s?type=public&per_page=100&page=%d", endpoint, page)
		req, err := http.NewRequestWithContext(cmd.Context(), "GET", url, nil)
		if err != nil {
			return nil, err
		}
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		req.Header.Set("Accept", "application/vnd.github+json")

		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("GitHub API request: %w", err)
		}

		if resp.StatusCode != 200 {
			resp.Body.Close()
			return nil, fmt.Errorf("GitHub API returned %d", resp.StatusCode)
		}

		var repos []struct {
			Name     string `json:"name"`
			FullName string `json:"full_name"`
			Fork     bool   `json:"fork"`
			Archived bool   `json:"archived"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&repos); err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("parse GitHub response: %w", err)
		}
		resp.Body.Close()

		if len(repos) == 0 {
			break
		}

		for _, r := range repos {
			if r.Fork && !includeForks {
				continue
			}
			if r.Archived && !includeArchived {
				continue
			}
			if filter != "" {
				matched, _ := filepath.Match(filter, r.Name)
				if !matched {
					continue
				}
			}
			id := strings.ToLower(r.Name)
			id = strings.ReplaceAll(id, ".", "-")
			allRepos = append(allRepos, importEntry{
				ID: id,
				Project: config.Project{
					Name:         r.Name,
					GitHubEvents: config.StringList{r.FullName},
				},
			})
		}

		page++
	}

	return allRepos, nil
}

func loadFromFile(path string) ([]importEntry, error) {
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

func loadFromJSON(path string) ([]importEntry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var items []struct {
		ID        string `json:"id"`
		Name      string `json:"name"`
		GitHub    string `json:"github"`
		PyPI      string `json:"pypi"`
		CRAN      string `json:"cran"`
		Homebrew  string `json:"homebrew"`
		Plausible string `json:"plausible"`
		OpenVSX   string `json:"openvsx"`
	}
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, fmt.Errorf("parse JSON: %w", err)
	}

	var entries []importEntry
	for _, item := range items {
		if item.ID == "" {
			return nil, fmt.Errorf("JSON entry missing 'id' field")
		}
		if !validIDRe.MatchString(item.ID) {
			return nil, fmt.Errorf("invalid project ID %q: must be lowercase alphanumeric with hyphens", item.ID)
		}
		name := item.Name
		if name == "" {
			name = item.ID
		}
		entries = append(entries, importEntry{
			ID: item.ID,
			Project: config.Project{
				Name:         name,
				GitHubEvents: toStringList(item.GitHub),
				PyPI:         toStringList(item.PyPI),
				CRAN:         toStringList(item.CRAN),
				Homebrew:     toStringList(item.Homebrew),
				Plausible:    toStringList(item.Plausible),
				OpenVSX:      toStringList(item.OpenVSX),
			},
		})
	}

	return entries, nil
}

func loadFromCSV(path string) ([]importEntry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

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

	var entries []importEntry
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
		if idx, ok := colIndex["github"]; ok && idx < len(row) {
			p.GitHubEvents = toStringList(row[idx])
		}
		if idx, ok := colIndex["pypi"]; ok && idx < len(row) {
			p.PyPI = toStringList(row[idx])
		}
		if idx, ok := colIndex["cran"]; ok && idx < len(row) {
			p.CRAN = toStringList(row[idx])
		}
		if idx, ok := colIndex["homebrew"]; ok && idx < len(row) {
			p.Homebrew = toStringList(row[idx])
		}
		if idx, ok := colIndex["plausible"]; ok && idx < len(row) {
			p.Plausible = toStringList(row[idx])
		}
		if idx, ok := colIndex["openvsx"]; ok && idx < len(row) {
			p.OpenVSX = toStringList(row[idx])
		}

		entries = append(entries, importEntry{ID: id, Project: p})
	}

	return entries, nil
}
