package source

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestOpenVSXFetch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/posit/shiny" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		rating := 4.5
		resp := map[string]interface{}{
			"downloadCount": 15000,
			"averageRating": rating,
			"reviewCount":   42,
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	o := &OpenVSX{
		Client:      server.Client(),
		ExtensionID: "posit/shiny",
		BaseURL:     server.URL,
	}

	records, err := o.Fetch(context.Background(), FetchOptions{
		ProjectID: "shiny",
		StartDate: time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}

	if len(records) != 3 {
		t.Fatalf("got %d records, want 3", len(records))
	}

	downloads := filterByMetric(records, "total_downloads")
	if len(downloads) != 1 || downloads[0].Value != 15000 {
		t.Errorf("total_downloads = %v, want 15000", downloads)
	}

	reviews := filterByMetric(records, "reviews")
	if len(reviews) != 1 || reviews[0].Value != 42 {
		t.Errorf("reviews = %v, want 42", reviews)
	}

	ratings := filterByMetric(records, "rating")
	if len(ratings) != 1 || ratings[0].Value != 450 {
		t.Errorf("rating = %v, want 450 (4.5 * 100)", ratings)
	}
}

func TestOpenVSXInvalidExtensionID(t *testing.T) {
	o := &OpenVSX{
		Client:      http.DefaultClient,
		ExtensionID: "invalid-no-slash",
	}

	_, err := o.Fetch(context.Background(), FetchOptions{
		ProjectID: "test",
		StartDate: time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC),
	})
	if err == nil {
		t.Fatal("expected error for invalid extension ID")
	}
}
