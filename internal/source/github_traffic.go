package source

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type GitHubTraffic struct {
	Client  *http.Client
	Token   string
	Repo    string
	BaseURL string
}

func (g *GitHubTraffic) Name() string { return "github-traffic" }

func (g *GitHubTraffic) baseURL() string {
	if g.BaseURL != "" {
		return g.BaseURL
	}
	return "https://api.github.com"
}

func (g *GitHubTraffic) Fetch(ctx context.Context, opts FetchOptions) ([]Record, error) {
	owner, repo := splitOwnerRepo(g.Repo)
	if owner == "" || repo == "" {
		return nil, fmt.Errorf("invalid github repo: %q", g.Repo)
	}

	var all []Record

	views, err := g.fetchViews(ctx, owner, repo, opts)
	if err != nil {
		return nil, fmt.Errorf("views: %w", err)
	}
	all = append(all, views...)

	clones, err := g.fetchClones(ctx, owner, repo, opts)
	if err != nil {
		return nil, fmt.Errorf("clones: %w", err)
	}
	all = append(all, clones...)

	return all, nil
}

func (g *GitHubTraffic) fetchViews(ctx context.Context, owner, repo string, opts FetchOptions) ([]Record, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/traffic/views?per=day", g.baseURL(), owner, repo)

	var result trafficResponse
	if err := g.get(ctx, url, &result); err != nil {
		return nil, err
	}

	var records []Record
	for _, item := range result.Items {
		t, err := time.Parse(time.RFC3339, item.Timestamp)
		if err != nil {
			continue
		}
		if t.Before(opts.StartDate) || !t.Before(opts.EndDate.AddDate(0, 0, 1)) {
			continue
		}
		date := t.Format("2006-01-02")
		records = append(records, Record{
			Metric:    "views",
			ProjectID: opts.ProjectID,
			Date:      date,
			Value:     item.Count,
		})
		records = append(records, Record{
			Metric:    "unique_views",
			ProjectID: opts.ProjectID,
			Date:      date,
			Value:     item.Uniques,
		})
	}
	return records, nil
}

func (g *GitHubTraffic) fetchClones(ctx context.Context, owner, repo string, opts FetchOptions) ([]Record, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/traffic/clones?per=day", g.baseURL(), owner, repo)

	var result trafficResponse
	if err := g.get(ctx, url, &result); err != nil {
		return nil, err
	}

	var records []Record
	for _, item := range result.Items {
		t, err := time.Parse(time.RFC3339, item.Timestamp)
		if err != nil {
			continue
		}
		if t.Before(opts.StartDate) || !t.Before(opts.EndDate.AddDate(0, 0, 1)) {
			continue
		}
		date := t.Format("2006-01-02")
		records = append(records, Record{
			Metric:    "clones",
			ProjectID: opts.ProjectID,
			Date:      date,
			Value:     item.Count,
		})
		records = append(records, Record{
			Metric:    "unique_clones",
			ProjectID: opts.ProjectID,
			Date:      date,
			Value:     item.Uniques,
		})
	}
	return records, nil
}

type trafficResponse struct {
	Count   int64         `json:"count"`
	Uniques int64         `json:"uniques"`
	Items   []trafficItem `json:"views,omitempty"`
}

type trafficItem struct {
	Timestamp string `json:"timestamp"`
	Count     int64  `json:"count"`
	Uniques   int64  `json:"uniques"`
}

func (r *trafficResponse) UnmarshalJSON(data []byte) error {
	type raw struct {
		Count   int64         `json:"count"`
		Uniques int64         `json:"uniques"`
		Views   []trafficItem `json:"views"`
		Clones  []trafficItem `json:"clones"`
	}
	var v raw
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	r.Count = v.Count
	r.Uniques = v.Uniques
	if v.Views != nil {
		r.Items = v.Views
	} else {
		r.Items = v.Clones
	}
	return nil
}

func (g *GitHubTraffic) get(ctx context.Context, url string, result interface{}) error {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}
	if g.Token != "" {
		req.Header.Set("Authorization", "Bearer "+g.Token)
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := g.Client.Do(req)
	if err != nil {
		return fmt.Errorf("request %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("github traffic API %s returned %d", req.URL.Path, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	return json.Unmarshal(body, result)
}
