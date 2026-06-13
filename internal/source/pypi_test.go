package source

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestPyPIFetch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"data": []map[string]interface{}{
				{"category": "without_mirrors", "date": "2025-06-01", "downloads": 1500},
				{"category": "without_mirrors", "date": "2025-06-02", "downloads": 2000},
				{"category": "with_mirrors", "date": "2025-06-01", "downloads": 3000},
				{"category": "without_mirrors", "date": "2025-05-31", "downloads": 1000},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := &PyPI{
		Client:  server.Client(),
		Package: "great-tables",
		BaseURL: server.URL,
	}

	records, err := p.Fetch(context.Background(), FetchOptions{
		ProjectID: "great-tables",
		StartDate: time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2025, 6, 2, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}

	if len(records) != 2 {
		t.Fatalf("got %d records, want 2", len(records))
	}

	if records[0].Value != 1500 {
		t.Errorf("records[0].Value = %d, want 1500", records[0].Value)
	}
	if records[1].Value != 2000 {
		t.Errorf("records[1].Value = %d, want 2000", records[1].Value)
	}
}

func TestPyPIAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	p := &PyPI{
		Client:  server.Client(),
		Package: "nonexistent",
		BaseURL: server.URL,
	}

	_, err := p.Fetch(context.Background(), FetchOptions{
		ProjectID: "nonexistent",
		StartDate: time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC),
	})
	if err == nil {
		t.Fatal("expected error for 404 response")
	}
}
