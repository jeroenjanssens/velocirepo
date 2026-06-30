package source

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Homebrew struct {
	Client  *http.Client
	Formula string
	BaseURL string // override for testing
}

func (h *Homebrew) Name() string { return "homebrew" }

func (h *Homebrew) baseURL() string {
	if h.BaseURL != "" {
		return h.BaseURL
	}
	return "https://formulae.brew.sh"
}

func (h *Homebrew) Fetch(ctx context.Context, opts FetchOptions) ([]Record, error) {
	url := fmt.Sprintf("%s/api/formula/%s.json", h.baseURL(), h.Formula)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := h.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request homebrew: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("homebrew API returned %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var result struct {
		Analytics struct {
			Install map[string]map[string]int64 `json:"install"`
		} `json:"analytics"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	today := time.Now().UTC().Format("2006-01-02")
	var records []Record

	for period, variants := range result.Analytics.Install {
		metric := "downloads_" + period
		var total int64
		for _, count := range variants {
			total += count
		}
		records = append(records, Record{
			Metric:    metric,
			ProjectID: opts.ProjectID,
			Target:    h.Formula,
			Date:      today,
			Value:     total,
		})
	}

	return records, nil
}

