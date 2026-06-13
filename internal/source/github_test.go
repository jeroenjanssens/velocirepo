package source

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestGitHubFetchStars(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/org/repo/stargazers", func(w http.ResponseWriter, r *http.Request) {
		page := r.URL.Query().Get("page")
		if page == "2" {
			w.Write([]byte("[]"))
			return
		}
		stars := []map[string]interface{}{
			{"starred_at": "2025-06-01T10:00:00Z", "user": map[string]string{"login": "alice"}},
			{"starred_at": "2025-06-01T12:00:00Z", "user": map[string]string{"login": "bob"}},
			{"starred_at": "2025-06-02T08:00:00Z", "user": map[string]string{"login": "carol"}},
		}
		json.NewEncoder(w).Encode(stars)
	})
	mux.HandleFunc("/repos/org/repo/forks", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("[]"))
	})
	mux.HandleFunc("/repos/org/repo/issues", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("[]"))
	})
	mux.HandleFunc("/repos/org/repo/pulls", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("[]"))
	})
	mux.HandleFunc("/repos/org/repo/issues/comments", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("[]"))
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	g := &GitHub{
		Client:  server.Client(),
		Token:   "test-token",
		Repo:    "org/repo",
		BaseURL: server.URL,
	}

	records, err := g.Fetch(context.Background(), FetchOptions{
		ProjectID: "myproject",
		StartDate: time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2025, 6, 2, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}

	// Should have stars counted by day
	starRecords := filterByMetric(records, "stars")
	if len(starRecords) != 2 {
		t.Fatalf("got %d star records, want 2 (one per day)", len(starRecords))
	}

	day1 := findByDate(starRecords, "2025-06-01")
	if day1 == nil {
		t.Fatal("no record for 2025-06-01")
	}
	if day1.Value != 2 {
		t.Errorf("2025-06-01 stars = %d, want 2", day1.Value)
	}

	day2 := findByDate(starRecords, "2025-06-02")
	if day2 == nil {
		t.Fatal("no record for 2025-06-02")
	}
	if day2.Value != 1 {
		t.Errorf("2025-06-02 stars = %d, want 1", day2.Value)
	}
}

func TestGitHubFetchIssuesAndPRs(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/org/repo/stargazers", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("[]"))
	})
	mux.HandleFunc("/repos/org/repo/forks", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("[]"))
	})
	mux.HandleFunc("/repos/org/repo/issues", func(w http.ResponseWriter, r *http.Request) {
		page := r.URL.Query().Get("page")
		if page == "2" {
			w.Write([]byte("[]"))
			return
		}
		issues := []map[string]interface{}{
			{"created_at": "2025-06-01T10:00:00Z", "closed_at": "2025-06-02T10:00:00Z", "user": map[string]string{"login": "alice"}},
			{"created_at": "2025-06-01T11:00:00Z", "closed_at": nil, "user": map[string]string{"login": "bob"}},
		}
		json.NewEncoder(w).Encode(issues)
	})
	mux.HandleFunc("/repos/org/repo/pulls", func(w http.ResponseWriter, r *http.Request) {
		page := r.URL.Query().Get("page")
		if page == "2" {
			w.Write([]byte("[]"))
			return
		}
		prs := []map[string]interface{}{
			{"created_at": "2025-06-01T09:00:00Z", "merged_at": "2025-06-01T15:00:00Z"},
		}
		json.NewEncoder(w).Encode(prs)
	})
	mux.HandleFunc("/repos/org/repo/issues/comments", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("[]"))
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	g := &GitHub{
		Client:  server.Client(),
		Token:   "test-token",
		Repo:    "org/repo",
		BaseURL: server.URL,
	}

	records, err := g.Fetch(context.Background(), FetchOptions{
		ProjectID: "myproject",
		StartDate: time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2025, 6, 2, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}

	opened := filterByMetric(records, "issues_opened")
	if len(opened) != 1 {
		t.Errorf("got %d issues_opened records, want 1", len(opened))
	} else if opened[0].Value != 2 {
		t.Errorf("issues_opened = %d, want 2", opened[0].Value)
	}

	closed := filterByMetric(records, "issues_closed")
	if len(closed) != 1 {
		t.Errorf("got %d issues_closed records, want 1", len(closed))
	} else if closed[0].Value != 1 {
		t.Errorf("issues_closed = %d, want 1", closed[0].Value)
	}

	prsOpened := filterByMetric(records, "prs_opened")
	if len(prsOpened) != 1 {
		t.Errorf("got %d prs_opened records, want 1", len(prsOpened))
	}

	prsMerged := filterByMetric(records, "prs_merged")
	if len(prsMerged) != 1 {
		t.Errorf("got %d prs_merged records, want 1", len(prsMerged))
	}
}

func TestGitHubAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	g := &GitHub{
		Client:  server.Client(),
		Token:   "bad-token",
		Repo:    "org/repo",
		BaseURL: server.URL,
	}

	_, err := g.Fetch(context.Background(), FetchOptions{
		ProjectID: "myproject",
		StartDate: time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC),
	})
	if err == nil {
		t.Fatal("expected error for 403 response")
	}
}

func TestGitHubContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("[]"))
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	g := &GitHub{
		Client:  server.Client(),
		Token:   "test-token",
		Repo:    "org/repo",
		BaseURL: server.URL,
	}

	_, err := g.Fetch(ctx, FetchOptions{
		ProjectID: "myproject",
		StartDate: time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC),
	})
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

func filterByMetric(records []Record, metric string) []Record {
	var result []Record
	for _, r := range records {
		if r.Metric == metric {
			result = append(result, r)
		}
	}
	return result
}

func findByDate(records []Record, date string) *Record {
	for _, r := range records {
		if r.Date == date {
			return &r
		}
	}
	return nil
}
