package source

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type OpenVSX struct {
	Client      *http.Client
	ExtensionID string
	BaseURL     string // override for testing
}

func (o *OpenVSX) Name() string { return "openvsx" }

func (o *OpenVSX) Fetch(ctx context.Context, opts FetchOptions) ([]Record, error) {
	parts := strings.SplitN(o.ExtensionID, "/", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid openvsx extension id: %q (expected namespace/extension)", o.ExtensionID)
	}

	baseURL := "https://open-vsx.org"
	if o.BaseURL != "" {
		baseURL = o.BaseURL
	}
	url := fmt.Sprintf("%s/api/%s/%s", baseURL, parts[0], parts[1])

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := o.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request openvsx: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("openvsx returned %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var result struct {
		DownloadCount int64    `json:"downloadCount"`
		AverageRating *float64 `json:"averageRating"`
		ReviewCount   int64    `json:"reviewCount"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	date := opts.EndDate.Format("2006-01-02")

	records := []Record{
		{Metric: "total_downloads", ProjectID: opts.ProjectID, Date: date, Value: result.DownloadCount},
		{Metric: "reviews", ProjectID: opts.ProjectID, Date: date, Value: result.ReviewCount},
	}

	if result.AverageRating != nil {
		// Store rating * 100 to preserve precision as int64
		records = append(records, Record{
			Metric:    "rating",
			ProjectID: opts.ProjectID,
			Date:      date,
			Value:     int64(*result.AverageRating * 100),
		})
	}

	return records, nil
}
