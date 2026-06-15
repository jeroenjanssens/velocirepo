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

func (g *GitHubEvents) Fetch(ctx context.Context, opts FetchOptions) ([]Record, error) {
	owner, repo := splitOwnerRepo(g.Repo)
	if owner == "" || repo == "" {
		return nil, fmt.Errorf("invalid github repo: %q", g.Repo)
	}

	url := fmt.Sprintf("%s/repos/%s/%s/events", g.baseURL(), owner, repo)

	// date -> metric -> count
	counts := make(map[string]map[string]int64)

	addCount := func(date, metric string) {
		if counts[date] == nil {
			counts[date] = make(map[string]int64)
		}
		counts[date][metric]++
	}

	page := 1
	maxPages := 10 // GitHub Events API hard limit is 10 pages
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

		var events []githubEvent
		if err := json.Unmarshal(body, &events); err != nil {
			return nil, fmt.Errorf("unmarshal events: %w", err)
		}

		if len(events) == 0 {
			break
		}

		allBeforeRange := true
		for _, evt := range events {
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

			date := t.Format("2006-01-02")
			metric := mapEventType(evt.Type, evt.Payload)
			if metric != "" {
				addCount(date, metric)
			}
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

	var records []Record
	for date, metrics := range counts {
		for metric, count := range metrics {
			records = append(records, Record{
				Metric:    metric,
				ProjectID: opts.ProjectID,
				Date:      date,
				Value:     count,
			})
		}
	}

	return records, nil
}

type githubEvent struct {
	Type      string          `json:"type"`
	CreatedAt string          `json:"created_at"`
	Payload   json.RawMessage `json:"payload"`
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

func mapEventType(eventType string, payload json.RawMessage) string {
	switch eventType {
	case "WatchEvent":
		return "stars"
	case "ForkEvent":
		return "forks"
	case "IssuesEvent":
		var p issuesPayload
		if json.Unmarshal(payload, &p) == nil {
			switch p.Action {
			case "opened":
				return "issues_opened"
			case "closed":
				return "issues_closed"
			}
		}
	case "PullRequestEvent":
		var p pullRequestPayload
		if json.Unmarshal(payload, &p) == nil {
			switch p.Action {
			case "opened":
				return "prs_opened"
			case "closed":
				if p.PullRequest.Merged {
					return "prs_merged"
				}
			}
		}
	case "PushEvent":
		return "pushes"
	case "ReleaseEvent":
		return "releases"
	case "IssueCommentEvent":
		return "comments"
	case "PullRequestReviewEvent":
		return "reviews"
	}
	return ""
}
