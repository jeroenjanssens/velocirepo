package source

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestPlausibleFetch(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("missing or wrong Authorization header")
		}

		callCount++
		var resp interface{}
		if callCount == 1 {
			resp = map[string]interface{}{
				"results": []map[string]interface{}{
					{"dimensions": []string{"2025-06-01"}, "metrics": []int64{100, 50, 60}},
					{"dimensions": []string{"2025-06-02"}, "metrics": []int64{200, 80, 90}},
				},
			}
		} else {
			resp = map[string]interface{}{
				"results": []map[string]interface{}{
					{"dimensions": []string{"2025-06-01", "/docs"}, "metrics": []int64{40, 20, 25}},
					{"dimensions": []string{"2025-06-01", "/blog"}, "metrics": []int64{60, 30, 35}},
					{"dimensions": []string{"2025-06-02", "/docs"}, "metrics": []int64{80, 40, 45}},
				},
			}
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := &Plausible{
		Client:  server.Client(),
		APIKey:  "test-key",
		SiteID:  "example.com",
		BaseURL: server.URL,
	}

	records, err := p.Fetch(context.Background(), FetchOptions{
		ProjectID: "mysite",
		StartDate: time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2025, 6, 2, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}

	// Site: 2 days * 3 metrics = 6
	// Pages: 3 rows * 3 metrics = 9
	if len(records) != 15 {
		t.Fatalf("got %d records, want 15", len(records))
	}

	sitePageviews := filterByMetric(records, "daily_site_pageviews")
	if len(sitePageviews) != 2 {
		t.Errorf("got %d site pageview records, want 2", len(sitePageviews))
	}

	siteVisitors := filterByMetric(records, "daily_site_visitors")
	if len(siteVisitors) != 2 {
		t.Errorf("got %d site visitor records, want 2", len(siteVisitors))
	}

	pagePageviews := filterByMetric(records, "daily_pageviews")
	if len(pagePageviews) != 3 {
		t.Errorf("got %d page pageview records, want 3", len(pagePageviews))
	}

	// Check page tags
	for _, r := range pagePageviews {
		if r.Tags["page"] == "" {
			t.Errorf("expected page tag, got empty")
		}
	}

	if callCount != 2 {
		t.Errorf("expected 2 API calls, got %d", callCount)
	}
}

func TestPlausibleAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	p := &Plausible{
		Client:  server.Client(),
		APIKey:  "bad-key",
		SiteID:  "example.com",
		BaseURL: server.URL,
	}

	_, err := p.Fetch(context.Background(), FetchOptions{
		ProjectID: "mysite",
		StartDate: time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC),
	})
	if err == nil {
		t.Fatal("expected error for 401 response")
	}
}
