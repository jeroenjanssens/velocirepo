package source

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestGitHubTrafficFetch(t *testing.T) {
	viewsResp := `{
		"count": 150,
		"uniques": 30,
		"views": [
			{"timestamp": "2025-06-01T00:00:00Z", "count": 50, "uniques": 10},
			{"timestamp": "2025-06-02T00:00:00Z", "count": 100, "uniques": 20}
		]
	}`
	clonesResp := `{
		"count": 25,
		"uniques": 5,
		"clones": [
			{"timestamp": "2025-06-01T00:00:00Z", "count": 10, "uniques": 2},
			{"timestamp": "2025-06-02T00:00:00Z", "count": 15, "uniques": 3}
		]
	}`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("per") != "day" {
			t.Errorf("expected per=day query param, got %q", r.URL.Query().Get("per"))
		}
		switch r.URL.Path {
		case "/repos/owner/repo/traffic/views":
			w.Write([]byte(viewsResp))
		case "/repos/owner/repo/traffic/clones":
			w.Write([]byte(clonesResp))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	g := &GitHubTraffic{
		Client:  srv.Client(),
		Token:   "test-token",
		Repo:    "owner/repo",
		BaseURL: srv.URL,
	}

	records, err := g.Fetch(context.Background(), FetchOptions{
		ProjectID: "my-project",
		StartDate: time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2025, 6, 2, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}

	if len(records) != 8 {
		t.Fatalf("got %d records, want 8", len(records))
	}

	expected := map[string]int64{
		"views:2025-06-01":         50,
		"unique_views:2025-06-01":  10,
		"views:2025-06-02":         100,
		"unique_views:2025-06-02":  20,
		"clones:2025-06-01":        10,
		"unique_clones:2025-06-01": 2,
		"clones:2025-06-02":        15,
		"unique_clones:2025-06-02": 3,
	}

	for _, r := range records {
		key := r.Metric + ":" + r.Date
		want, ok := expected[key]
		if !ok {
			t.Errorf("unexpected record: %s = %d", key, r.Value)
			continue
		}
		if r.Value != want {
			t.Errorf("%s = %d, want %d", key, r.Value, want)
		}
		if r.ProjectID != "my-project" {
			t.Errorf("ProjectID = %q, want %q", r.ProjectID, "my-project")
		}
		delete(expected, key)
	}

	for key := range expected {
		t.Errorf("missing record: %s", key)
	}
}

func TestGitHubTrafficDateFiltering(t *testing.T) {
	viewsResp := `{
		"count": 200,
		"uniques": 40,
		"views": [
			{"timestamp": "2025-05-30T00:00:00Z", "count": 10, "uniques": 2},
			{"timestamp": "2025-06-01T00:00:00Z", "count": 50, "uniques": 10},
			{"timestamp": "2025-06-05T00:00:00Z", "count": 80, "uniques": 15}
		]
	}`
	clonesResp := `{"count": 0, "uniques": 0, "clones": []}`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/owner/repo/traffic/views":
			w.Write([]byte(viewsResp))
		case "/repos/owner/repo/traffic/clones":
			w.Write([]byte(clonesResp))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	g := &GitHubTraffic{
		Client:  srv.Client(),
		Repo:    "owner/repo",
		BaseURL: srv.URL,
	}

	records, err := g.Fetch(context.Background(), FetchOptions{
		ProjectID: "test",
		StartDate: time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2025, 6, 3, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}

	// Should only include 2025-06-01 (within range), not 05-30 (before) or 06-05 (after)
	if len(records) != 2 {
		t.Fatalf("got %d records, want 2 (views + unique_views for 06-01)", len(records))
	}

	if records[0].Date != "2025-06-01" {
		t.Errorf("Date = %q, want 2025-06-01", records[0].Date)
	}
}

func TestGitHubTrafficInvalidRepo(t *testing.T) {
	g := &GitHubTraffic{
		Client: http.DefaultClient,
		Repo:   "invalid",
	}

	_, err := g.Fetch(context.Background(), FetchOptions{
		ProjectID: "test",
		StartDate: time.Now(),
		EndDate:   time.Now(),
	})
	if err == nil {
		t.Fatal("expected error for invalid repo")
	}
}

func TestGitHubTrafficAuthHeader(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Write([]byte(`{"count": 0, "uniques": 0, "views": []}`))
	}))
	defer srv.Close()

	g := &GitHubTraffic{
		Client:  srv.Client(),
		Token:   "my-secret-token",
		Repo:    "owner/repo",
		BaseURL: srv.URL,
	}

	g.Fetch(context.Background(), FetchOptions{
		ProjectID: "test",
		StartDate: time.Now().AddDate(0, 0, -7),
		EndDate:   time.Now(),
	})

	if gotAuth != "Bearer my-secret-token" {
		t.Errorf("Authorization = %q, want %q", gotAuth, "Bearer my-secret-token")
	}
}
