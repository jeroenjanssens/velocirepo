package source

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
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
		jsonHandler(t, http.StatusOK, resp)(w, r)
	}))
	defer server.Close()

	o := &OpenVSX{
		Client:      server.Client(),
		ExtensionID: "posit/shiny",
		BaseURL:     server.URL,
	}

	records, err := o.Fetch(context.Background(), juneFetchOptions("shiny", 1, 1))
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}

	assertRecordCount(t, records, 3)
	assertMetricValues(t, records, map[string]int64{
		"total_downloads": 15000,
		"total_reviews":   42,
		"total_ratings":   450,
	})
}

func TestOpenVSXInvalidExtensionID(t *testing.T) {
	o := &OpenVSX{
		Client:      http.DefaultClient,
		ExtensionID: "invalid-no-slash",
	}

	_, err := o.Fetch(context.Background(), juneFetchOptions("test", 1, 1))
	if err == nil {
		t.Fatal("expected error for invalid extension ID")
	}
}
