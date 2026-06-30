package source

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCRANFetch(t *testing.T) {
	server := httptest.NewServer(jsonHandler(t, http.StatusOK, []map[string]interface{}{
		{
			"package": "dplyr",
			"downloads": []map[string]interface{}{
				{"day": "2025-06-01", "downloads": 5000},
				{"day": "2025-06-02", "downloads": 6000},
			},
		},
	}))
	defer server.Close()

	c := &CRAN{
		Client:  server.Client(),
		Package: "dplyr",
		BaseURL: server.URL,
	}

	records, err := c.Fetch(context.Background(), juneFetchOptions("dplyr", 1, 2))
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}

	assertRecordCount(t, records, 2)
	if records[0].Value != 5000 {
		t.Errorf("records[0].Value = %d, want 5000", records[0].Value)
	}
	if records[1].Metric != "daily_downloads" {
		t.Errorf("records[1].Metric = %q, want %q", records[1].Metric, "daily_downloads")
	}
}

func TestCRANAPIError(t *testing.T) {
	server := errorServer(t, http.StatusInternalServerError)

	c := &CRAN{
		Client:  server.Client(),
		Package: "nonexistent",
		BaseURL: server.URL,
	}

	_, err := c.Fetch(context.Background(), juneFetchOptions("nonexistent", 1, 1))
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
}
