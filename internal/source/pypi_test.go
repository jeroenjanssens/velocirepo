package source

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPyPIFetch(t *testing.T) {
	server := httptest.NewServer(jsonHandler(t, http.StatusOK, map[string]interface{}{
		"data": []map[string]interface{}{
			{"category": "without_mirrors", "date": "2025-06-01", "downloads": 1500},
			{"category": "without_mirrors", "date": "2025-06-02", "downloads": 2000},
			{"category": "with_mirrors", "date": "2025-06-01", "downloads": 3000},
			{"category": "without_mirrors", "date": "2025-05-31", "downloads": 1000},
		},
	}))
	defer server.Close()

	p := &PyPI{
		Client:  server.Client(),
		Package: "great-tables",
		BaseURL: server.URL,
	}

	records, err := p.Fetch(context.Background(), juneFetchOptions("great-tables", 1, 2))
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}

	assertRecordCount(t, records, 2)

	if records[0].Value != 1500 {
		t.Errorf("records[0].Value = %d, want 1500", records[0].Value)
	}
	if records[1].Value != 2000 {
		t.Errorf("records[1].Value = %d, want 2000", records[1].Value)
	}
}

func TestPyPIAPIError(t *testing.T) {
	server := errorServer(t, http.StatusNotFound)

	p := &PyPI{
		Client:  server.Client(),
		Package: "nonexistent",
		BaseURL: server.URL,
	}

	_, err := p.Fetch(context.Background(), juneFetchOptions("nonexistent", 1, 1))
	if err == nil {
		t.Fatal("expected error for 404 response")
	}
}
