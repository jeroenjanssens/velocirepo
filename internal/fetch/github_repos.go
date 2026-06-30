package fetch

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/jeroenjanssens/velocirepo/internal/config"
)

type ImportEntry struct {
	ID      string
	Project config.Project
}

type ImportOptions struct {
	IncludeForks    bool
	IncludeArchived bool
	Filter          string
}

func FetchGitHubRepos(ctx context.Context, token string, endpoint string, opts ImportOptions) ([]ImportEntry, error) {
	client := &http.Client{Timeout: 30 * time.Second}

	var allRepos []ImportEntry
	page := 1

	for {
		url := fmt.Sprintf("https://api.github.com/%s?type=public&per_page=100&page=%d", endpoint, page)
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
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
			_ = resp.Body.Close()
			return nil, fmt.Errorf("GitHub API returned %d", resp.StatusCode)
		}

		var repos []struct {
			Name     string `json:"name"`
			FullName string `json:"full_name"`
			Fork     bool   `json:"fork"`
			Archived bool   `json:"archived"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&repos); err != nil {
			_ = resp.Body.Close()
			return nil, fmt.Errorf("parse GitHub response: %w", err)
		}
		_ = resp.Body.Close()

		if len(repos) == 0 {
			break
		}

		for _, r := range repos {
			if r.Fork && !opts.IncludeForks {
				continue
			}
			if r.Archived && !opts.IncludeArchived {
				continue
			}
			if opts.Filter != "" {
				matched, _ := filepath.Match(opts.Filter, r.Name)
				if !matched {
					continue
				}
			}
			id := strings.ToLower(r.Name)
			id = strings.ReplaceAll(id, ".", "-")
			allRepos = append(allRepos, ImportEntry{
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
