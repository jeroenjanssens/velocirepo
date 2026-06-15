package source

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestGitHubEventsFetch(t *testing.T) {
	eventsResp := `[
		{"type": "WatchEvent", "created_at": "2025-06-10T10:00:00Z", "payload": {}},
		{"type": "WatchEvent", "created_at": "2025-06-10T11:00:00Z", "payload": {}},
		{"type": "ForkEvent", "created_at": "2025-06-10T12:00:00Z", "payload": {}},
		{"type": "IssuesEvent", "created_at": "2025-06-10T13:00:00Z", "payload": {"action": "opened"}},
		{"type": "IssuesEvent", "created_at": "2025-06-10T14:00:00Z", "payload": {"action": "closed"}},
		{"type": "PullRequestEvent", "created_at": "2025-06-10T15:00:00Z", "payload": {"action": "opened", "pull_request": {"merged": false}}},
		{"type": "PullRequestEvent", "created_at": "2025-06-10T16:00:00Z", "payload": {"action": "closed", "pull_request": {"merged": true}}},
		{"type": "PushEvent", "created_at": "2025-06-10T17:00:00Z", "payload": {}},
		{"type": "ReleaseEvent", "created_at": "2025-06-10T18:00:00Z", "payload": {}},
		{"type": "IssueCommentEvent", "created_at": "2025-06-10T19:00:00Z", "payload": {}},
		{"type": "PullRequestReviewEvent", "created_at": "2025-06-10T20:00:00Z", "payload": {}}
	]`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/owner/repo/events" {
			http.NotFound(w, r)
			return
		}
		if r.URL.Query().Get("page") == "2" {
			w.Write([]byte("[]"))
			return
		}
		w.Write([]byte(eventsResp))
	}))
	defer srv.Close()

	g := &GitHubEvents{
		Client:  srv.Client(),
		Token:   "test-token",
		Repo:    "owner/repo",
		BaseURL: srv.URL,
	}

	records, err := g.Fetch(context.Background(), FetchOptions{
		ProjectID: "my-project",
		StartDate: time.Date(2025, 6, 10, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2025, 6, 10, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}

	expected := map[string]int64{
		"stars":          2,
		"forks":          1,
		"issues_opened":  1,
		"issues_closed":  1,
		"prs_opened":     1,
		"prs_merged":     1,
		"pushes":         1,
		"releases":       1,
		"comments":       1,
		"reviews":        1,
	}

	if len(records) != len(expected) {
		t.Fatalf("got %d records, want %d", len(records), len(expected))
	}

	got := make(map[string]int64)
	for _, r := range records {
		got[r.Metric] = r.Value
		if r.ProjectID != "my-project" {
			t.Errorf("ProjectID = %q, want %q", r.ProjectID, "my-project")
		}
		if r.Date != "2025-06-10" {
			t.Errorf("Date = %q, want %q", r.Date, "2025-06-10")
		}
	}

	for metric, want := range expected {
		if got[metric] != want {
			t.Errorf("%s = %d, want %d", metric, got[metric], want)
		}
	}
}

func TestGitHubEventsDateFiltering(t *testing.T) {
	eventsResp := `[
		{"type": "WatchEvent", "created_at": "2025-06-12T10:00:00Z", "payload": {}},
		{"type": "WatchEvent", "created_at": "2025-06-10T10:00:00Z", "payload": {}},
		{"type": "WatchEvent", "created_at": "2025-06-08T10:00:00Z", "payload": {}}
	]`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("page") == "2" {
			w.Write([]byte("[]"))
			return
		}
		w.Write([]byte(eventsResp))
	}))
	defer srv.Close()

	g := &GitHubEvents{
		Client:  srv.Client(),
		Repo:    "owner/repo",
		BaseURL: srv.URL,
	}

	records, err := g.Fetch(context.Background(), FetchOptions{
		ProjectID: "test",
		StartDate: time.Date(2025, 6, 9, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2025, 6, 11, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}

	if len(records) != 1 {
		t.Fatalf("got %d records, want 1 (only 2025-06-10 in range)", len(records))
	}
	if records[0].Date != "2025-06-10" {
		t.Errorf("Date = %q, want 2025-06-10", records[0].Date)
	}
}

func TestGitHubEventsIgnoresUnknownTypes(t *testing.T) {
	eventsResp := `[
		{"type": "CreateEvent", "created_at": "2025-06-10T10:00:00Z", "payload": {}},
		{"type": "DeleteEvent", "created_at": "2025-06-10T11:00:00Z", "payload": {}},
		{"type": "WatchEvent", "created_at": "2025-06-10T12:00:00Z", "payload": {}}
	]`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("page") == "2" {
			w.Write([]byte("[]"))
			return
		}
		w.Write([]byte(eventsResp))
	}))
	defer srv.Close()

	g := &GitHubEvents{
		Client:  srv.Client(),
		Repo:    "owner/repo",
		BaseURL: srv.URL,
	}

	records, err := g.Fetch(context.Background(), FetchOptions{
		ProjectID: "test",
		StartDate: time.Date(2025, 6, 10, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2025, 6, 10, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}

	if len(records) != 1 {
		t.Fatalf("got %d records, want 1 (only WatchEvent mapped)", len(records))
	}
	if records[0].Metric != "stars" {
		t.Errorf("Metric = %q, want stars", records[0].Metric)
	}
}

func TestGitHubEventsInvalidRepo(t *testing.T) {
	g := &GitHubEvents{
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

func TestGitHubEventsAuthHeader(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Write([]byte("[]"))
	}))
	defer srv.Close()

	g := &GitHubEvents{
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

func TestGitHubEventsPRClosedNotMerged(t *testing.T) {
	eventsResp := `[
		{"type": "PullRequestEvent", "created_at": "2025-06-10T10:00:00Z", "payload": {"action": "closed", "pull_request": {"merged": false}}}
	]`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("page") == "2" {
			w.Write([]byte("[]"))
			return
		}
		w.Write([]byte(eventsResp))
	}))
	defer srv.Close()

	g := &GitHubEvents{
		Client:  srv.Client(),
		Repo:    "owner/repo",
		BaseURL: srv.URL,
	}

	records, err := g.Fetch(context.Background(), FetchOptions{
		ProjectID: "test",
		StartDate: time.Date(2025, 6, 10, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2025, 6, 10, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}

	if len(records) != 0 {
		t.Fatalf("got %d records, want 0 (closed but not merged should be skipped)", len(records))
	}
}

func TestGitHubEventsMultipleDays(t *testing.T) {
	eventsResp := `[
		{"type": "WatchEvent", "created_at": "2025-06-11T10:00:00Z", "payload": {}},
		{"type": "WatchEvent", "created_at": "2025-06-11T11:00:00Z", "payload": {}},
		{"type": "WatchEvent", "created_at": "2025-06-10T10:00:00Z", "payload": {}},
		{"type": "ForkEvent", "created_at": "2025-06-10T12:00:00Z", "payload": {}}
	]`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("page") == "2" {
			w.Write([]byte("[]"))
			return
		}
		w.Write([]byte(eventsResp))
	}))
	defer srv.Close()

	g := &GitHubEvents{
		Client:  srv.Client(),
		Repo:    "owner/repo",
		BaseURL: srv.URL,
	}

	records, err := g.Fetch(context.Background(), FetchOptions{
		ProjectID: "test",
		StartDate: time.Date(2025, 6, 10, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2025, 6, 11, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}

	got := make(map[string]int64)
	for _, r := range records {
		got[r.Date+":"+r.Metric] = r.Value
	}

	if got["2025-06-11:stars"] != 2 {
		t.Errorf("2025-06-11 stars = %d, want 2", got["2025-06-11:stars"])
	}
	if got["2025-06-10:stars"] != 1 {
		t.Errorf("2025-06-10 stars = %d, want 1", got["2025-06-10:stars"])
	}
	if got["2025-06-10:forks"] != 1 {
		t.Errorf("2025-06-10 forks = %d, want 1", got["2025-06-10:forks"])
	}
}
