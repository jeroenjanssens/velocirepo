package source

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type CRAN struct {
	Client  *http.Client
	Package string
	BaseURL string // override for testing
}

func (c *CRAN) Name() string { return "cran" }

func (c *CRAN) baseURL() string {
	if c.BaseURL != "" {
		return c.BaseURL
	}
	return "https://cranlogs.r-pkg.org"
}

func (c *CRAN) Fetch(ctx context.Context, opts FetchOptions) ([]Record, error) {
	startStr := opts.StartDate.Format("2006-01-02")
	endStr := opts.EndDate.Format("2006-01-02")
	url := fmt.Sprintf("%s/downloads/daily/%s:%s/%s", c.baseURL(), startStr, endStr, c.Package)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request cranlogs: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("cranlogs returned %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var result []struct {
		Downloads []struct {
			Day       string `json:"day"`
			Downloads int64  `json:"downloads"`
		} `json:"downloads"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	if len(result) == 0 {
		return nil, nil
	}

	var records []Record
	for _, entry := range result[0].Downloads {
		entryDate, err := time.Parse("2006-01-02", entry.Day)
		if err != nil {
			continue
		}
		if entryDate.Before(opts.StartDate) || !entryDate.Before(opts.EndDate.AddDate(0, 0, 1)) {
			continue
		}
		records = append(records, Record{
			Metric:    "downloads",
			ProjectID: opts.ProjectID,
			Target:    c.Package,
			Date:      entry.Day,
			Value:     entry.Downloads,
		})
	}

	return records, nil
}
