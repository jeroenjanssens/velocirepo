package source

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCRANFetch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := []map[string]interface{}{
			{
				"package": "dplyr",
				"downloads": []map[string]interface{}{
					{"day": "2025-06-01", "downloads": 5000},
					{"day": "2025-06-02", "downloads": 6000},
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	c := &CRAN{
		Client:  server.Client(),
		Package: "dplyr",
		BaseURL: server.URL,
	}

	records, err := c.Fetch(context.Background(), FetchOptions{
		ProjectID: "dplyr",
		StartDate: time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2025, 6, 2, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}

	if len(records) != 2 {
		t.Fatalf("got %d records, want 2", len(records))
	}
	if records[0].Value != 5000 {
		t.Errorf("records[0].Value = %d, want 5000", records[0].Value)
	}
	if records[1].Metric != "downloads" {
		t.Errorf("records[1].Metric = %q, want %q", records[1].Metric, "downloads")
	}
}

func TestCRANAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	c := &CRAN{
		Client:  server.Client(),
		Package: "nonexistent",
		BaseURL: server.URL,
	}

	_, err := c.Fetch(context.Background(), FetchOptions{
		ProjectID: "nonexistent",
		StartDate: time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC),
	})
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
}
