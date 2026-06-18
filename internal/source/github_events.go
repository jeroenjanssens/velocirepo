package source

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type GitHubEvents struct {
	Client  *http.Client
	Token   string
	Repo    string
	BaseURL string
}

func (g *GitHubEvents) Name() string { return "github" }

func (g *GitHubEvents) graphqlURL() string {
	if g.BaseURL != "" {
		return g.BaseURL
	}
	return "https://api.github.com/graphql"
}

func (g *GitHubEvents) FetchEvents(ctx context.Context, opts FetchOptions) ([]GitHubEvent, error) {
	owner, repo := splitOwnerRepo(g.Repo)
	if owner == "" || repo == "" {
		return nil, fmt.Errorf("invalid github repo: %q", g.Repo)
	}

	var events []GitHubEvent

	stars, err := g.fetchStargazers(ctx, owner, repo, opts)
	if err != nil {
		return nil, fmt.Errorf("stargazers: %w", err)
	}
	events = append(events, stars...)

	forks, err := g.fetchForks(ctx, owner, repo, opts)
	if err != nil {
		return nil, fmt.Errorf("forks: %w", err)
	}
	events = append(events, forks...)

	issues, err := g.fetchIssues(ctx, owner, repo, opts)
	if err != nil {
		return nil, fmt.Errorf("issues: %w", err)
	}
	events = append(events, issues...)

	prs, err := g.fetchPullRequests(ctx, owner, repo, opts)
	if err != nil {
		return nil, fmt.Errorf("pull requests: %w", err)
	}
	events = append(events, prs...)

	return events, nil
}

func (g *GitHubEvents) fetchStargazers(ctx context.Context, owner, repo string, opts FetchOptions) ([]GitHubEvent, error) {
	query := `query($owner: String!, $name: String!, $after: String) {
		repository(owner: $owner, name: $name) {
			stargazers(first: 100, after: $after, orderBy: {field: STARRED_AT, direction: DESC}) {
				edges {
					starredAt
					node { login }
				}
				pageInfo { hasNextPage endCursor }
			}
		}
	}`

	var events []GitHubEvent
	var cursor *string

	for {
		vars := map[string]interface{}{"owner": owner, "name": repo, "after": cursor}
		resp, err := g.doGraphQL(ctx, query, vars)
		if err != nil {
			return nil, err
		}

		var result struct {
			Data struct {
				Repository struct {
					Stargazers struct {
						Edges []struct {
							StarredAt string `json:"starredAt"`
							Node      struct {
								Login string `json:"login"`
							} `json:"node"`
						} `json:"edges"`
						PageInfo pageInfo `json:"pageInfo"`
					} `json:"stargazers"`
				} `json:"repository"`
			} `json:"data"`
		}
		if err := json.Unmarshal(resp, &result); err != nil {
			return nil, fmt.Errorf("unmarshal stargazers: %w", err)
		}

		done := false
		for _, edge := range result.Data.Repository.Stargazers.Edges {
			t, err := time.Parse(time.RFC3339, edge.StarredAt)
			if err != nil {
				continue
			}
			if t.Before(opts.StartDate) {
				done = true
				break
			}
			if !t.Before(opts.EndDate.AddDate(0, 0, 1)) {
				continue
			}
			events = append(events, GitHubEvent{
				EventType:  "star",
				ProjectID:  opts.ProjectID,
				GitHubRepo: g.Repo,
				Datetime:   edge.StarredAt,
				User:       edge.Node.Login,
			})
		}

		pi := result.Data.Repository.Stargazers.PageInfo
		if done || !pi.HasNextPage {
			break
		}
		cursor = &pi.EndCursor
	}

	return events, nil
}

func (g *GitHubEvents) fetchForks(ctx context.Context, owner, repo string, opts FetchOptions) ([]GitHubEvent, error) {
	query := `query($owner: String!, $name: String!, $after: String) {
		repository(owner: $owner, name: $name) {
			forks(first: 100, after: $after, orderBy: {field: CREATED_AT, direction: DESC}) {
				nodes {
					createdAt
					owner { login }
				}
				pageInfo { hasNextPage endCursor }
			}
		}
	}`

	var events []GitHubEvent
	var cursor *string

	for {
		vars := map[string]interface{}{"owner": owner, "name": repo, "after": cursor}
		resp, err := g.doGraphQL(ctx, query, vars)
		if err != nil {
			return nil, err
		}

		var result struct {
			Data struct {
				Repository struct {
					Forks struct {
						Nodes []struct {
							CreatedAt string `json:"createdAt"`
							Owner     struct {
								Login string `json:"login"`
							} `json:"owner"`
						} `json:"nodes"`
						PageInfo pageInfo `json:"pageInfo"`
					} `json:"forks"`
				} `json:"repository"`
			} `json:"data"`
		}
		if err := json.Unmarshal(resp, &result); err != nil {
			return nil, fmt.Errorf("unmarshal forks: %w", err)
		}

		done := false
		for _, node := range result.Data.Repository.Forks.Nodes {
			t, err := time.Parse(time.RFC3339, node.CreatedAt)
			if err != nil {
				continue
			}
			if t.Before(opts.StartDate) {
				done = true
				break
			}
			if !t.Before(opts.EndDate.AddDate(0, 0, 1)) {
				continue
			}
			events = append(events, GitHubEvent{
				EventType:  "fork",
				ProjectID:  opts.ProjectID,
				GitHubRepo: g.Repo,
				Datetime:   node.CreatedAt,
				User:       node.Owner.Login,
			})
		}

		pi := result.Data.Repository.Forks.PageInfo
		if done || !pi.HasNextPage {
			break
		}
		cursor = &pi.EndCursor
	}

	return events, nil
}

func (g *GitHubEvents) fetchIssues(ctx context.Context, owner, repo string, opts FetchOptions) ([]GitHubEvent, error) {
	query := `query($owner: String!, $name: String!, $after: String) {
		repository(owner: $owner, name: $name) {
			issues(first: 100, after: $after, orderBy: {field: CREATED_AT, direction: DESC}) {
				nodes {
					createdAt
					closedAt
					author { login }
				}
				pageInfo { hasNextPage endCursor }
			}
		}
	}`

	var events []GitHubEvent
	var cursor *string

	for {
		vars := map[string]interface{}{"owner": owner, "name": repo, "after": cursor}
		resp, err := g.doGraphQL(ctx, query, vars)
		if err != nil {
			return nil, err
		}

		var result struct {
			Data struct {
				Repository struct {
					Issues struct {
						Nodes []struct {
							CreatedAt string  `json:"createdAt"`
							ClosedAt  *string `json:"closedAt"`
							Author    *struct {
								Login string `json:"login"`
							} `json:"author"`
						} `json:"nodes"`
						PageInfo pageInfo `json:"pageInfo"`
					} `json:"issues"`
				} `json:"repository"`
			} `json:"data"`
		}
		if err := json.Unmarshal(resp, &result); err != nil {
			return nil, fmt.Errorf("unmarshal issues: %w", err)
		}

		done := false
		for _, node := range result.Data.Repository.Issues.Nodes {
			t, err := time.Parse(time.RFC3339, node.CreatedAt)
			if err != nil {
				continue
			}
			if t.Before(opts.StartDate) {
				done = true
				break
			}

			user := ""
			if node.Author != nil {
				user = node.Author.Login
			}

			if !t.Before(opts.EndDate.AddDate(0, 0, 1)) {
				continue
			}
			events = append(events, GitHubEvent{
				EventType:  "issue_open",
				ProjectID:  opts.ProjectID,
				GitHubRepo: g.Repo,
				Datetime:   node.CreatedAt,
				User:       user,
			})

			if node.ClosedAt != nil {
				ct, err := time.Parse(time.RFC3339, *node.ClosedAt)
				if err == nil && !ct.Before(opts.StartDate) && ct.Before(opts.EndDate.AddDate(0, 0, 1)) {
					events = append(events, GitHubEvent{
						EventType:  "issue_close",
						ProjectID:  opts.ProjectID,
						GitHubRepo: g.Repo,
						Datetime:   *node.ClosedAt,
						User:       user,
					})
				}
			}
		}

		pi := result.Data.Repository.Issues.PageInfo
		if done || !pi.HasNextPage {
			break
		}
		cursor = &pi.EndCursor
	}

	return events, nil
}

func (g *GitHubEvents) fetchPullRequests(ctx context.Context, owner, repo string, opts FetchOptions) ([]GitHubEvent, error) {
	query := `query($owner: String!, $name: String!, $after: String) {
		repository(owner: $owner, name: $name) {
			pullRequests(first: 100, after: $after, orderBy: {field: CREATED_AT, direction: DESC}) {
				nodes {
					createdAt
					closedAt
					mergedAt
					author { login }
				}
				pageInfo { hasNextPage endCursor }
			}
		}
	}`

	var events []GitHubEvent
	var cursor *string

	for {
		vars := map[string]interface{}{"owner": owner, "name": repo, "after": cursor}
		resp, err := g.doGraphQL(ctx, query, vars)
		if err != nil {
			return nil, err
		}

		var result struct {
			Data struct {
				Repository struct {
					PullRequests struct {
						Nodes []struct {
							CreatedAt string  `json:"createdAt"`
							ClosedAt  *string `json:"closedAt"`
							MergedAt  *string `json:"mergedAt"`
							Author    *struct {
								Login string `json:"login"`
							} `json:"author"`
						} `json:"nodes"`
						PageInfo pageInfo `json:"pageInfo"`
					} `json:"pullRequests"`
				} `json:"repository"`
			} `json:"data"`
		}
		if err := json.Unmarshal(resp, &result); err != nil {
			return nil, fmt.Errorf("unmarshal pull requests: %w", err)
		}

		done := false
		for _, node := range result.Data.Repository.PullRequests.Nodes {
			t, err := time.Parse(time.RFC3339, node.CreatedAt)
			if err != nil {
				continue
			}
			if t.Before(opts.StartDate) {
				done = true
				break
			}

			user := ""
			if node.Author != nil {
				user = node.Author.Login
			}

			if !t.Before(opts.EndDate.AddDate(0, 0, 1)) {
				continue
			}
			events = append(events, GitHubEvent{
				EventType:  "pr_open",
				ProjectID:  opts.ProjectID,
				GitHubRepo: g.Repo,
				Datetime:   node.CreatedAt,
				User:       user,
			})

			if node.MergedAt != nil {
				mt, err := time.Parse(time.RFC3339, *node.MergedAt)
				if err == nil && !mt.Before(opts.StartDate) && mt.Before(opts.EndDate.AddDate(0, 0, 1)) {
					events = append(events, GitHubEvent{
						EventType:  "pr_merge",
						ProjectID:  opts.ProjectID,
						GitHubRepo: g.Repo,
						Datetime:   *node.MergedAt,
						User:       user,
					})
				}
			}
		}

		pi := result.Data.Repository.PullRequests.PageInfo
		if done || !pi.HasNextPage {
			break
		}
		cursor = &pi.EndCursor
	}

	return events, nil
}

type pageInfo struct {
	HasNextPage bool   `json:"hasNextPage"`
	EndCursor   string `json:"endCursor"`
}

type graphqlRequest struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables"`
}

func (g *GitHubEvents) doGraphQL(ctx context.Context, query string, vars map[string]interface{}) ([]byte, error) {
	body, err := json.Marshal(graphqlRequest{Query: query, Variables: vars})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", g.graphqlURL(), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	if g.Token != "" {
		req.Header.Set("Authorization", "Bearer "+g.Token)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := g.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("graphql request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("graphql returned %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read graphql response: %w", err)
	}

	var errResp struct {
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if json.Unmarshal(data, &errResp) == nil && len(errResp.Errors) > 0 {
		return nil, fmt.Errorf("graphql error: %s", errResp.Errors[0].Message)
	}

	return data, nil
}
