package source

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHomebrewFetch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"analytics": map[string]interface{}{
				"install": map[string]interface{}{
					"30d": map[string]int64{
						"wget":        22297,
						"wget --HEAD": 33,
					},
					"90d": map[string]int64{
						"wget":        74263,
						"wget --HEAD": 159,
					},
					"365d": map[string]int64{
						"wget":        382652,
						"wget --HEAD": 722,
					},
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	h := &Homebrew{
		Client:  server.Client(),
		Formula: "wget",
		BaseURL: server.URL,
	}

	records, err := h.Fetch(context.Background(), FetchOptions{
		ProjectID: "wget",
		StartDate: time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}

	if len(records) != 3 {
		t.Fatalf("got %d records, want 3", len(records))
	}

	metrics := make(map[string]int64)
	for _, r := range records {
		metrics[r.Metric] = r.Value
		if r.ProjectID != "wget" {
			t.Errorf("expected project_id=wget, got %s", r.ProjectID)
		}
	}

	if metrics["downloads_30d"] != 22330 {
		t.Errorf("downloads_30d = %d, want 22330", metrics["downloads_30d"])
	}
	if metrics["downloads_90d"] != 74422 {
		t.Errorf("downloads_90d = %d, want 74422", metrics["downloads_90d"])
	}
	if metrics["downloads_365d"] != 383374 {
		t.Errorf("downloads_365d = %d, want 383374", metrics["downloads_365d"])
	}
}

func TestHomebrewAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	h := &Homebrew{
		Client:  server.Client(),
		Formula: "nonexistent",
		BaseURL: server.URL,
	}

	_, err := h.Fetch(context.Background(), FetchOptions{
		ProjectID: "nonexistent",
		StartDate: time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC),
	})
	if err == nil {
		t.Fatal("expected error for 404 response")
	}
}
