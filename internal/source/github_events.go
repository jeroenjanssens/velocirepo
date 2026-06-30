package source

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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

func (g *GitHubEvents) FetchEvents(ctx context.Context, opts FetchOptions) ([]Event, error) {
	owner, repo := splitOwnerRepo(g.Repo)
	if owner == "" || repo == "" {
		return nil, fmt.Errorf("invalid github repo: %q", g.Repo)
	}

	var events []Event

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

type githubPageReader func([]byte) ([]Event, pageInfo, bool, error)

func (g *GitHubEvents) fetchPaginatedEvents(ctx context.Context, owner, repo, query string, readPage githubPageReader) ([]Event, error) {
	var events []Event
	var cursor *string

	for {
		vars := map[string]interface{}{"owner": owner, "name": repo, "after": cursor}
		resp, err := g.doGraphQL(ctx, query, vars)
		if err != nil {
			return nil, err
		}

		pageEvents, pi, done, err := readPage(resp)
		if err != nil {
			return nil, err
		}
		events = append(events, pageEvents...)

		if done || !pi.HasNextPage {
			break
		}
		cursor = &pi.EndCursor
	}

	return events, nil
}

func (g *GitHubEvents) fetchStargazers(ctx context.Context, owner, repo string, opts FetchOptions) ([]Event, error) {
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

	return g.fetchPaginatedEvents(ctx, owner, repo, query, func(resp []byte) ([]Event, pageInfo, bool, error) {
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
			return nil, pageInfo{}, false, fmt.Errorf("unmarshal stargazers: %w", err)
		}

		var events []Event
		done := false
		for _, edge := range result.Data.Repository.Stargazers.Edges {
			include, stop := includeGitHubEventTime(edge.StarredAt, opts)
			if stop {
				done = true
				break
			}
			if !include {
				continue
			}
			events = append(events, githubEvent(opts, g.Repo, "star", edge.StarredAt, userTags(edge.Node.Login)))
		}

		return events, result.Data.Repository.Stargazers.PageInfo, done, nil
	})
}

func (g *GitHubEvents) fetchForks(ctx context.Context, owner, repo string, opts FetchOptions) ([]Event, error) {
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

	return g.fetchPaginatedEvents(ctx, owner, repo, query, func(resp []byte) ([]Event, pageInfo, bool, error) {
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
			return nil, pageInfo{}, false, fmt.Errorf("unmarshal forks: %w", err)
		}

		var events []Event
		done := false
		for _, node := range result.Data.Repository.Forks.Nodes {
			include, stop := includeGitHubEventTime(node.CreatedAt, opts)
			if stop {
				done = true
				break
			}
			if !include {
				continue
			}
			events = append(events, githubEvent(opts, g.Repo, "fork", node.CreatedAt, userTags(node.Owner.Login)))
		}

		return events, result.Data.Repository.Forks.PageInfo, done, nil
	})
}

func (g *GitHubEvents) fetchIssues(ctx context.Context, owner, repo string, opts FetchOptions) ([]Event, error) {
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

	return g.fetchPaginatedEvents(ctx, owner, repo, query, func(resp []byte) ([]Event, pageInfo, bool, error) {
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
			return nil, pageInfo{}, false, fmt.Errorf("unmarshal issues: %w", err)
		}

		var events []Event
		done := false
		for _, node := range result.Data.Repository.Issues.Nodes {
			include, stop := includeGitHubEventTime(node.CreatedAt, opts)
			if stop {
				done = true
				break
			}

			var login string
			if node.Author != nil && node.Author.Login != "" {
				login = node.Author.Login
			}
			tags := userTags(login)

			if !include {
				continue
			}
			events = append(events, githubEvent(opts, g.Repo, "issue_open", node.CreatedAt, tags))

			if node.ClosedAt != nil {
				events = appendOptionalGitHubEvent(events, opts, g.Repo, "issue_close", *node.ClosedAt, tags)
			}
		}

		return events, result.Data.Repository.Issues.PageInfo, done, nil
	})
}

func (g *GitHubEvents) fetchPullRequests(ctx context.Context, owner, repo string, opts FetchOptions) ([]Event, error) {
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

	return g.fetchPaginatedEvents(ctx, owner, repo, query, func(resp []byte) ([]Event, pageInfo, bool, error) {
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
			return nil, pageInfo{}, false, fmt.Errorf("unmarshal pull requests: %w", err)
		}

		var events []Event
		done := false
		for _, node := range result.Data.Repository.PullRequests.Nodes {
			include, stop := includeGitHubEventTime(node.CreatedAt, opts)
			if stop {
				done = true
				break
			}

			var login string
			if node.Author != nil && node.Author.Login != "" {
				login = node.Author.Login
			}
			tags := userTags(login)

			if !include {
				continue
			}
			events = append(events, githubEvent(opts, g.Repo, "pr_open", node.CreatedAt, tags))

			if node.MergedAt != nil {
				events = appendOptionalGitHubEvent(events, opts, g.Repo, "pr_merge", *node.MergedAt, tags)
			}
		}

		return events, result.Data.Repository.PullRequests.PageInfo, done, nil
	})
}

func includeGitHubEventTime(timestamp string, opts FetchOptions) (include bool, stop bool) {
	t, err := time.Parse(time.RFC3339, timestamp)
	if err != nil {
		return false, false
	}
	if t.Before(opts.StartDate) {
		return false, true
	}
	return inDateRange(t, opts.StartDate, opts.EndDate), false
}

func appendOptionalGitHubEvent(events []Event, opts FetchOptions, repo, eventType, timestamp string, tags map[string]string) []Event {
	t, err := time.Parse(time.RFC3339, timestamp)
	if err != nil || !inDateRange(t, opts.StartDate, opts.EndDate) {
		return events
	}
	return append(events, githubEvent(opts, repo, eventType, timestamp, tags))
}

func githubEvent(opts FetchOptions, repo, eventType, timestamp string, tags map[string]string) Event {
	return Event{
		Type:      eventType,
		ProjectID: opts.ProjectID,
		Target:    repo,
		Datetime:  timestamp,
		Tags:      tags,
	}
}

func userTags(login string) map[string]string {
	if login == "" {
		return nil
	}
	return map[string]string{"user": login}
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

	headers := map[string]string{"Content-Type": "application/json"}
	if g.Token != "" {
		headers["Authorization"] = "Bearer " + g.Token
	}
	data, err := doRequest(ctx, g.Client, httpJSONRequest{
		Method:       http.MethodPost,
		URL:          g.graphqlURL(),
		Headers:      headers,
		Body:         bytes.NewReader(body),
		RequestError: "graphql request",
		StatusError:  "graphql returned",
	})
	if err != nil {
		return nil, err
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
