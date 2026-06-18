package source

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type Plausible struct {
	Client  *http.Client
	APIKey  string
	SiteID  string
	BaseURL string // override for testing
}

func (p *Plausible) Name() string { return "plausible" }

func (p *Plausible) Fetch(ctx context.Context, opts FetchOptions) ([]Record, error) {
	payload := map[string]interface{}{
		"site_id": p.SiteID,
		"metrics": []string{"pageviews", "visitors", "visits"},
		"date_range": []string{
			opts.StartDate.Format("2006-01-02"),
			opts.EndDate.Format("2006-01-02"),
		},
		"dimensions": []string{"time:day"},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	baseURL := "https://plausible.io"
	if p.BaseURL != "" {
		baseURL = p.BaseURL
	}

	req, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/api/v2/query", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+p.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request plausible: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("plausible returned %d", resp.StatusCode)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var result struct {
		Results []struct {
			Dimensions []string `json:"dimensions"`
			Metrics    []int64  `json:"metrics"`
		} `json:"results"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	metricNames := []string{"pageviews", "visitors", "visits"}
	var records []Record
	for _, row := range result.Results {
		if len(row.Dimensions) == 0 || len(row.Metrics) < 3 {
			continue
		}
		date := row.Dimensions[0]
		for i, name := range metricNames {
			records = append(records, Record{
				Metric:    name,
				ProjectID: opts.ProjectID,
				Target:    p.SiteID,
				Date:      date,
				Value:     row.Metrics[i],
			})
		}
	}

	return records, nil
}
