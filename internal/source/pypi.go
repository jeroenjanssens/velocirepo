package source

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type PyPI struct {
	Client  *http.Client
	Package string
	BaseURL string // override for testing
}

func (p *PyPI) Name() string { return "pypi" }

func (p *PyPI) baseURL() string {
	if p.BaseURL != "" {
		return p.BaseURL
	}
	return "https://pypistats.org"
}

func (p *PyPI) Fetch(ctx context.Context, opts FetchOptions) ([]Record, error) {
	url := fmt.Sprintf("%s/api/packages/%s/overall", p.baseURL(), p.Package)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	q := req.URL.Query()
	q.Set("mirrors", "false")
	req.URL.RawQuery = q.Encode()

	resp, err := p.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request pypistats: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("pypistats returned %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var result struct {
		Data []struct {
			Category  string `json:"category"`
			Date      string `json:"date"`
			Downloads int64  `json:"downloads"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	var records []Record
	for _, entry := range result.Data {
		if entry.Category == "with_mirrors" {
			continue
		}
		entryDate, err := time.Parse("2006-01-02", entry.Date)
		if err != nil {
			continue
		}
		if entryDate.Before(opts.StartDate) || !entryDate.Before(opts.EndDate.AddDate(0, 0, 1)) {
			continue
		}
		records = append(records, Record{
			Metric:    "downloads",
			ProjectID: opts.ProjectID,
			Date:      entry.Date,
			Value:     entry.Downloads,
		})
	}

	return records, nil
}
