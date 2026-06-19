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
	siteRecords, err := p.fetchSiteMetrics(ctx, opts)
	if err != nil {
		return nil, err
	}

	pageRecords, err := p.fetchPageMetrics(ctx, opts)
	if err != nil {
		return nil, err
	}

	return append(siteRecords, pageRecords...), nil
}

func (p *Plausible) fetchSiteMetrics(ctx context.Context, opts FetchOptions) ([]Record, error) {
	const pageSize = 10000

	metricNames := []string{"daily_site_pageviews", "daily_site_visitors", "daily_site_visits"}
	var records []Record
	offset := 0

	for {
		payload := map[string]interface{}{
			"site_id": p.SiteID,
			"metrics": []string{"pageviews", "visitors", "visits"},
			"date_range": []string{
				opts.StartDate.Format("2006-01-02"),
				opts.EndDate.Format("2006-01-02"),
			},
			"dimensions": []string{"time:day"},
			"pagination": map[string]int{"limit": pageSize, "offset": offset},
			"include":    map[string]bool{"total_rows": true},
		}

		var result queryResult
		if err := p.query(ctx, payload, &result); err != nil {
			return nil, err
		}

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

		offset += pageSize
		if result.Meta.TotalRows == 0 || offset >= result.Meta.TotalRows {
			break
		}
	}

	return records, nil
}

func (p *Plausible) fetchPageMetrics(ctx context.Context, opts FetchOptions) ([]Record, error) {
	const pageSize = 10000

	metricNames := []string{"daily_pageviews", "daily_visitors", "daily_visits"}
	var records []Record
	offset := 0

	for {
		payload := map[string]interface{}{
			"site_id": p.SiteID,
			"metrics": []string{"pageviews", "visitors", "visits"},
			"date_range": []string{
				opts.StartDate.Format("2006-01-02"),
				opts.EndDate.Format("2006-01-02"),
			},
			"dimensions": []string{"time:day", "event:page"},
			"pagination": map[string]int{"limit": pageSize, "offset": offset},
			"include":    map[string]bool{"total_rows": true},
		}

		var result queryResult
		if err := p.query(ctx, payload, &result); err != nil {
			return nil, err
		}

		for _, row := range result.Results {
			if len(row.Dimensions) < 2 || len(row.Metrics) < 3 {
				continue
			}
			date := row.Dimensions[0]
			page := row.Dimensions[1]
			for i, name := range metricNames {
				records = append(records, Record{
					Metric:    name,
					ProjectID: opts.ProjectID,
					Target:    p.SiteID,
					Date:      date,
					Value:     row.Metrics[i],
					Tags:      map[string]string{"page": page},
				})
			}
		}

		offset += pageSize
		if result.Meta.TotalRows == 0 || offset >= result.Meta.TotalRows {
			break
		}
	}

	return records, nil
}

type queryResult struct {
	Results []struct {
		Dimensions []string `json:"dimensions"`
		Metrics    []int64  `json:"metrics"`
	} `json:"results"`
	Meta struct {
		TotalRows int `json:"total_rows"`
	} `json:"meta"`
}

func (p *Plausible) query(ctx context.Context, payload interface{}, result interface{}) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	baseURL := "https://plausible.io"
	if p.BaseURL != "" {
		baseURL = p.BaseURL
	}

	req, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/api/v2/query", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+p.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.Client.Do(req)
	if err != nil {
		return fmt.Errorf("request plausible: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("plausible returned %d", resp.StatusCode)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if err := json.Unmarshal(respBody, result); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}
	return nil
}
