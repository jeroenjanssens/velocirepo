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
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("missing or wrong Authorization header")
		}

		resp := map[string]interface{}{
			"results": []map[string]interface{}{
				{"dimensions": []string{"2025-06-01"}, "metrics": []int64{100, 50, 60}},
				{"dimensions": []string{"2025-06-02"}, "metrics": []int64{200, 80, 90}},
			},
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

	// 2 days * 3 metrics = 6 records
	if len(records) != 6 {
		t.Fatalf("got %d records, want 6", len(records))
	}

	pageviews := filterByMetric(records, "pageviews")
	if len(pageviews) != 2 {
		t.Errorf("got %d pageview records, want 2", len(pageviews))
	}

	visitors := filterByMetric(records, "visitors")
	if len(visitors) != 2 {
		t.Errorf("got %d visitor records, want 2", len(visitors))
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
