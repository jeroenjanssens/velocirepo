package source

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
)

type GitHubEvents struct {
	Client  *http.Client
	Token   string
	Repo    string
	BaseURL string
}

func (g *GitHubEvents) Name() string { return "github-events" }

func (g *GitHubEvents) baseURL() string {
	if g.BaseURL != "" {
		return g.BaseURL
	}
	return "https://api.github.com"
}

func (g *GitHubEvents) FetchEvents(ctx context.Context, opts FetchOptions) ([]GitHubEvent, error) {
	owner, repo := splitOwnerRepo(g.Repo)
	if owner == "" || repo == "" {
		return nil, fmt.Errorf("invalid github repo: %q", g.Repo)
	}

	url := fmt.Sprintf("%s/repos/%s/%s/events", g.baseURL(), owner, repo)

	var events []GitHubEvent

	page := 1
	maxPages := 10
	hitCeiling := false

	for page <= maxPages {
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return nil, err
		}

		if g.Token != "" {
			req.Header.Set("Authorization", "Bearer "+g.Token)
		}
		req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

		q := req.URL.Query()
		q.Set("page", fmt.Sprintf("%d", page))
		q.Set("per_page", "100")
		req.URL.RawQuery = q.Encode()

		resp, err := g.Client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("request %s: %w", req.URL, err)
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return nil, fmt.Errorf("github events API %s returned %d", req.URL.Path, resp.StatusCode)
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("read response body: %w", err)
		}

		var apiEvents []githubEvent
		if err := json.Unmarshal(body, &apiEvents); err != nil {
			return nil, fmt.Errorf("unmarshal events: %w", err)
		}

		if len(apiEvents) == 0 {
			break
		}

		allBeforeRange := true
		for _, evt := range apiEvents {
			t, err := parseGitHubTime(evt.CreatedAt)
			if err != nil {
				continue
			}

			if t.Before(opts.StartDate) {
				continue
			}
			allBeforeRange = false

			if !t.Before(opts.EndDate.AddDate(0, 0, 1)) {
				continue
			}

			eventType := mapToEventType(evt.Type, evt.Payload)
			if eventType == "" {
				continue
			}

			events = append(events, GitHubEvent{
				EventType:  eventType,
				ProjectID:  opts.ProjectID,
				GitHubRepo: g.Repo,
				Datetime:   evt.CreatedAt,
				User:       evt.Actor.Login,
			})
		}

		if allBeforeRange {
			break
		}

		if page == maxPages {
			hitCeiling = true
		}
		page++
	}

	if hitCeiling {
		slog.Warn("github events: hit 10-page API limit, some events may be missing", "repo", g.Repo)
	}

	return events, nil
}

type githubEvent struct {
	Type      string          `json:"type"`
	CreatedAt string          `json:"created_at"`
	Payload   json.RawMessage `json:"payload"`
	Actor     struct {
		Login string `json:"login"`
	} `json:"actor"`
}

type issuesPayload struct {
	Action string `json:"action"`
}

type pullRequestPayload struct {
	Action      string `json:"action"`
	PullRequest struct {
		Merged bool `json:"merged"`
	} `json:"pull_request"`
}

func mapToEventType(eventType string, payload json.RawMessage) string {
	switch eventType {
	case "WatchEvent":
		return "star"
	case "ForkEvent":
		return "fork"
	case "IssuesEvent":
		var p issuesPayload
		if json.Unmarshal(payload, &p) == nil {
			switch p.Action {
			case "opened":
				return "issue_open"
			case "closed":
				return "issue_close"
			}
		}
	case "IssueCommentEvent":
		return "issue_comment"
	case "PullRequestEvent":
		var p pullRequestPayload
		if json.Unmarshal(payload, &p) == nil {
			switch p.Action {
			case "opened":
				return "pr_open"
			case "closed":
				if p.PullRequest.Merged {
					return "pr_merge"
				}
			}
		}
	case "PullRequestReviewCommentEvent":
		return "pr_comment"
	}
	return ""
}
