package source

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type GitHub struct {
	Client  *http.Client
	Token   string
	Repo    string
	BaseURL string // override for testing; defaults to https://api.github.com
}

func (g *GitHub) Name() string { return "github" }

func (g *GitHub) baseURL() string {
	if g.BaseURL != "" {
		return g.BaseURL
	}
	return "https://api.github.com"
}

func (g *GitHub) Fetch(ctx context.Context, opts FetchOptions) ([]Record, error) {
	owner, repo := splitOwnerRepo(g.Repo)
	if owner == "" || repo == "" {
		return nil, fmt.Errorf("invalid github repo: %q", g.Repo)
	}

	var all []Record

	counts, err := g.fetchStars(ctx, owner, repo, opts)
	if err != nil {
		return nil, fmt.Errorf("stars: %w", err)
	}
	all = append(all, counts...)

	counts, err = g.fetchForks(ctx, owner, repo, opts)
	if err != nil {
		return nil, fmt.Errorf("forks: %w", err)
	}
	all = append(all, counts...)

	counts, err = g.fetchIssues(ctx, owner, repo, opts)
	if err != nil {
		return nil, fmt.Errorf("issues: %w", err)
	}
	all = append(all, counts...)

	counts, err = g.fetchPulls(ctx, owner, repo, opts)
	if err != nil {
		return nil, fmt.Errorf("pulls: %w", err)
	}
	all = append(all, counts...)

	counts, err = g.fetchComments(ctx, owner, repo, opts)
	if err != nil {
		return nil, fmt.Errorf("comments: %w", err)
	}
	all = append(all, counts...)

	return all, nil
}

func (g *GitHub) fetchStars(ctx context.Context, owner, repo string, opts FetchOptions) ([]Record, error) {
	counts := make(map[string]int64)
	url := fmt.Sprintf("%s/repos/%s/%s/stargazers", g.baseURL(), owner, repo)

	err := g.paginate(ctx, url, map[string]string{
		"Accept": "application/vnd.github.star+json",
	}, map[string]string{"direction": "desc"}, func(data []byte) (bool, error) {
		var items []struct {
			StarredAt string `json:"starred_at"`
		}
		if err := json.Unmarshal(data, &items); err != nil {
			return false, err
		}
		if len(items) == 0 {
			return false, nil
		}

		for _, item := range items {
			t, err := parseGitHubTime(item.StarredAt)
			if err != nil {
				continue
			}
			if !t.Before(opts.EndDate.AddDate(0, 0, 1)) {
				continue
			}
			if t.Before(opts.StartDate) {
				return false, nil
			}
			counts[t.Format("2006-01-02")]++
		}
		return true, nil
	})
	if err != nil {
		return nil, err
	}

	return countsToRecords(counts, opts.ProjectID, "stars"), nil
}

func (g *GitHub) fetchForks(ctx context.Context, owner, repo string, opts FetchOptions) ([]Record, error) {
	counts := make(map[string]int64)
	url := fmt.Sprintf("%s/repos/%s/%s/forks", g.baseURL(), owner, repo)

	err := g.paginate(ctx, url, nil, map[string]string{"sort": "newest"}, func(data []byte) (bool, error) {
		var items []struct {
			CreatedAt string `json:"created_at"`
		}
		if err := json.Unmarshal(data, &items); err != nil {
			return false, err
		}
		if len(items) == 0 {
			return false, nil
		}

		for _, item := range items {
			t, err := parseGitHubTime(item.CreatedAt)
			if err != nil {
				continue
			}
			if t.Before(opts.StartDate) {
				return false, nil
			}
			if !t.Before(opts.EndDate.AddDate(0, 0, 1)) {
				continue
			}
			counts[t.Format("2006-01-02")]++
		}
		return true, nil
	})
	if err != nil {
		return nil, err
	}

	return countsToRecords(counts, opts.ProjectID, "forks"), nil
}

func (g *GitHub) fetchIssues(ctx context.Context, owner, repo string, opts FetchOptions) ([]Record, error) {
	openCounts := make(map[string]int64)
	closeCounts := make(map[string]int64)
	url := fmt.Sprintf("%s/repos/%s/%s/issues", g.baseURL(), owner, repo)

	err := g.paginate(ctx, url, nil, map[string]string{
		"state": "all", "sort": "created", "direction": "desc",
	}, func(data []byte) (bool, error) {
		var items []struct {
			CreatedAt   string `json:"created_at"`
			ClosedAt    *string `json:"closed_at"`
			PullRequest *struct{} `json:"pull_request"`
		}
		if err := json.Unmarshal(data, &items); err != nil {
			return false, err
		}
		if len(items) == 0 {
			return false, nil
		}

		for _, item := range items {
			if item.PullRequest != nil {
				continue
			}
			ct, err := parseGitHubTime(item.CreatedAt)
			if err != nil {
				continue
			}
			if ct.Before(opts.StartDate) {
				return false, nil
			}
			if inRange(ct, opts) {
				openCounts[ct.Format("2006-01-02")]++
			}
			if item.ClosedAt != nil {
				if clt, err := parseGitHubTime(*item.ClosedAt); err == nil && inRange(clt, opts) {
					closeCounts[clt.Format("2006-01-02")]++
				}
			}
		}
		return true, nil
	})
	if err != nil {
		return nil, err
	}

	var records []Record
	records = append(records, countsToRecords(openCounts, opts.ProjectID, "issues_opened")...)
	records = append(records, countsToRecords(closeCounts, opts.ProjectID, "issues_closed")...)
	return records, nil
}

func (g *GitHub) fetchPulls(ctx context.Context, owner, repo string, opts FetchOptions) ([]Record, error) {
	openCounts := make(map[string]int64)
	mergeCounts := make(map[string]int64)
	url := fmt.Sprintf("%s/repos/%s/%s/pulls", g.baseURL(), owner, repo)

	err := g.paginate(ctx, url, nil, map[string]string{
		"state": "all", "sort": "created", "direction": "desc",
	}, func(data []byte) (bool, error) {
		var items []struct {
			CreatedAt string  `json:"created_at"`
			MergedAt  *string `json:"merged_at"`
		}
		if err := json.Unmarshal(data, &items); err != nil {
			return false, err
		}
		if len(items) == 0 {
			return false, nil
		}

		for _, item := range items {
			ct, err := parseGitHubTime(item.CreatedAt)
			if err != nil {
				continue
			}
			if ct.Before(opts.StartDate) {
				return false, nil
			}
			if inRange(ct, opts) {
				openCounts[ct.Format("2006-01-02")]++
			}
			if item.MergedAt != nil {
				if mt, err := parseGitHubTime(*item.MergedAt); err == nil && inRange(mt, opts) {
					mergeCounts[mt.Format("2006-01-02")]++
				}
			}
		}
		return true, nil
	})
	if err != nil {
		return nil, err
	}

	var records []Record
	records = append(records, countsToRecords(openCounts, opts.ProjectID, "prs_opened")...)
	records = append(records, countsToRecords(mergeCounts, opts.ProjectID, "prs_merged")...)
	return records, nil
}

func (g *GitHub) fetchComments(ctx context.Context, owner, repo string, opts FetchOptions) ([]Record, error) {
	counts := make(map[string]int64)
	url := fmt.Sprintf("%s/repos/%s/%s/issues/comments", g.baseURL(), owner, repo)

	err := g.paginate(ctx, url, nil, map[string]string{
		"sort": "created", "direction": "desc",
	}, func(data []byte) (bool, error) {
		var items []struct {
			CreatedAt string `json:"created_at"`
		}
		if err := json.Unmarshal(data, &items); err != nil {
			return false, err
		}
		if len(items) == 0 {
			return false, nil
		}

		for _, item := range items {
			ct, err := parseGitHubTime(item.CreatedAt)
			if err != nil {
				continue
			}
			if ct.Before(opts.StartDate) {
				return false, nil
			}
			if inRange(ct, opts) {
				counts[ct.Format("2006-01-02")]++
			}
		}
		return true, nil
	})
	if err != nil {
		return nil, err
	}

	return countsToRecords(counts, opts.ProjectID, "comments"), nil
}

const maxPages = 1000

func (g *GitHub) paginate(ctx context.Context, baseURL string, headers map[string]string, params map[string]string, fn func([]byte) (bool, error)) error {
	page := 1
	perPage := "100"

	for page <= maxPages {
		req, err := http.NewRequestWithContext(ctx, "GET", baseURL, nil)
		if err != nil {
			return err
		}

		if g.Token != "" {
			req.Header.Set("Authorization", "Bearer "+g.Token)
		}
		req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
		for k, v := range headers {
			req.Header.Set(k, v)
		}

		q := req.URL.Query()
		q.Set("page", fmt.Sprintf("%d", page))
		q.Set("per_page", perPage)
		for k, v := range params {
			q.Set(k, v)
		}
		req.URL.RawQuery = q.Encode()

		resp, err := g.Client.Do(req)
		if err != nil {
			return fmt.Errorf("request %s: %w", req.URL, err)
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return fmt.Errorf("github API %s returned %d", req.URL.Path, resp.StatusCode)
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return fmt.Errorf("read response body: %w", err)
		}

		cont, err := fn(body)
		if err != nil {
			return err
		}
		if !cont {
			return nil
		}

		page++
	}
	return fmt.Errorf("pagination exceeded %d pages for %s", maxPages, baseURL)
}


func parseGitHubTime(s string) (time.Time, error) {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t, err = time.Parse("2006-01-02T15:04:05Z", s)
	}
	return t, err
}

func inRange(t time.Time, opts FetchOptions) bool {
	return !t.Before(opts.StartDate) && t.Before(opts.EndDate.AddDate(0, 0, 1))
}

func countsToRecords(counts map[string]int64, projectID, metric string) []Record {
	var records []Record
	for date, count := range counts {
		records = append(records, Record{
			Metric:    metric,
			ProjectID: projectID,
			Date:      date,
			Value:     count,
		})
	}
	return records
}

func splitOwnerRepo(repo string) (string, string) {
	for i, c := range repo {
		if c == '/' {
			return repo[:i], repo[i+1:]
		}
	}
	return "", ""
}
