package source

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestGitHubEventsFetchEvents(t *testing.T) {
	eventsResp := `[
		{"type": "WatchEvent", "created_at": "2025-06-10T10:00:00Z", "payload": {}, "actor": {"login": "alice"}},
		{"type": "WatchEvent", "created_at": "2025-06-10T11:00:00Z", "payload": {}, "actor": {"login": "bob"}},
		{"type": "ForkEvent", "created_at": "2025-06-10T12:00:00Z", "payload": {}, "actor": {"login": "carol"}},
		{"type": "IssuesEvent", "created_at": "2025-06-10T13:00:00Z", "payload": {"action": "opened"}, "actor": {"login": "dave"}},
		{"type": "IssuesEvent", "created_at": "2025-06-10T14:00:00Z", "payload": {"action": "closed"}, "actor": {"login": "eve"}},
		{"type": "IssueCommentEvent", "created_at": "2025-06-10T15:00:00Z", "payload": {}, "actor": {"login": "frank"}},
		{"type": "PullRequestEvent", "created_at": "2025-06-10T16:00:00Z", "payload": {"action": "opened", "pull_request": {"merged": false}}, "actor": {"login": "grace"}},
		{"type": "PullRequestEvent", "created_at": "2025-06-10T17:00:00Z", "payload": {"action": "closed", "pull_request": {"merged": true}}, "actor": {"login": "heidi"}},
		{"type": "PullRequestReviewCommentEvent", "created_at": "2025-06-10T18:00:00Z", "payload": {}, "actor": {"login": "ivan"}}
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

	events, err := g.FetchEvents(context.Background(), FetchOptions{
		ProjectID: "my-project",
		StartDate: time.Date(2025, 6, 10, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2025, 6, 10, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("FetchEvents failed: %v", err)
	}

	if len(events) != 9 {
		t.Fatalf("got %d events, want 9", len(events))
	}

	expected := []struct {
		eventType string
		user      string
	}{
		{"star", "alice"},
		{"star", "bob"},
		{"fork", "carol"},
		{"issue_open", "dave"},
		{"issue_close", "eve"},
		{"issue_comment", "frank"},
		{"pr_open", "grace"},
		{"pr_merge", "heidi"},
		{"pr_comment", "ivan"},
	}

	for i, want := range expected {
		if events[i].EventType != want.eventType {
			t.Errorf("events[%d].EventType = %q, want %q", i, events[i].EventType, want.eventType)
		}
		if events[i].User != want.user {
			t.Errorf("events[%d].User = %q, want %q", i, events[i].User, want.user)
		}
		if events[i].ProjectID != "my-project" {
			t.Errorf("events[%d].ProjectID = %q, want %q", i, events[i].ProjectID, "my-project")
		}
		if events[i].GitHubRepo != "owner/repo" {
			t.Errorf("events[%d].GitHubRepo = %q, want %q", i, events[i].GitHubRepo, "owner/repo")
		}
	}
}

func TestGitHubEventsDateFiltering(t *testing.T) {
	eventsResp := `[
		{"type": "WatchEvent", "created_at": "2025-06-12T10:00:00Z", "payload": {}, "actor": {"login": "alice"}},
		{"type": "WatchEvent", "created_at": "2025-06-10T10:00:00Z", "payload": {}, "actor": {"login": "bob"}},
		{"type": "WatchEvent", "created_at": "2025-06-08T10:00:00Z", "payload": {}, "actor": {"login": "carol"}}
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

	events, err := g.FetchEvents(context.Background(), FetchOptions{
		ProjectID: "test",
		StartDate: time.Date(2025, 6, 9, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2025, 6, 11, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("FetchEvents failed: %v", err)
	}

	if len(events) != 1 {
		t.Fatalf("got %d events, want 1 (only 2025-06-10 in range)", len(events))
	}
	if events[0].Datetime != "2025-06-10T10:00:00Z" {
		t.Errorf("Datetime = %q, want 2025-06-10T10:00:00Z", events[0].Datetime)
	}
}

func TestGitHubEventsIgnoresUnknownTypes(t *testing.T) {
	eventsResp := `[
		{"type": "CreateEvent", "created_at": "2025-06-10T10:00:00Z", "payload": {}, "actor": {"login": "a"}},
		{"type": "DeleteEvent", "created_at": "2025-06-10T11:00:00Z", "payload": {}, "actor": {"login": "b"}},
		{"type": "WatchEvent", "created_at": "2025-06-10T12:00:00Z", "payload": {}, "actor": {"login": "c"}}
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

	events, err := g.FetchEvents(context.Background(), FetchOptions{
		ProjectID: "test",
		StartDate: time.Date(2025, 6, 10, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2025, 6, 10, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("FetchEvents failed: %v", err)
	}

	if len(events) != 1 {
		t.Fatalf("got %d events, want 1 (only WatchEvent mapped)", len(events))
	}
	if events[0].EventType != "star" {
		t.Errorf("EventType = %q, want star", events[0].EventType)
	}
}

func TestGitHubEventsInvalidRepo(t *testing.T) {
	g := &GitHubEvents{
		Client: http.DefaultClient,
		Repo:   "invalid",
	}

	_, err := g.FetchEvents(context.Background(), FetchOptions{
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

	g.FetchEvents(context.Background(), FetchOptions{
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
		{"type": "PullRequestEvent", "created_at": "2025-06-10T10:00:00Z", "payload": {"action": "closed", "pull_request": {"merged": false}}, "actor": {"login": "alice"}}
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

	events, err := g.FetchEvents(context.Background(), FetchOptions{
		ProjectID: "test",
		StartDate: time.Date(2025, 6, 10, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2025, 6, 10, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("FetchEvents failed: %v", err)
	}

	if len(events) != 0 {
		t.Fatalf("got %d events, want 0 (closed but not merged should be skipped)", len(events))
	}
}

func TestGitHubEventsMultipleDays(t *testing.T) {
	eventsResp := `[
		{"type": "WatchEvent", "created_at": "2025-06-11T10:00:00Z", "payload": {}, "actor": {"login": "alice"}},
		{"type": "WatchEvent", "created_at": "2025-06-11T11:00:00Z", "payload": {}, "actor": {"login": "bob"}},
		{"type": "WatchEvent", "created_at": "2025-06-10T10:00:00Z", "payload": {}, "actor": {"login": "carol"}},
		{"type": "ForkEvent", "created_at": "2025-06-10T12:00:00Z", "payload": {}, "actor": {"login": "dave"}}
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

	events, err := g.FetchEvents(context.Background(), FetchOptions{
		ProjectID: "test",
		StartDate: time.Date(2025, 6, 10, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2025, 6, 11, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("FetchEvents failed: %v", err)
	}

	if len(events) != 4 {
		t.Fatalf("got %d events, want 4", len(events))
	}

	day10 := 0
	day11 := 0
	for _, e := range events {
		if e.Datetime[:10] == "2025-06-10" {
			day10++
		} else if e.Datetime[:10] == "2025-06-11" {
			day11++
		}
	}

	if day10 != 2 {
		t.Errorf("day 10 events = %d, want 2", day10)
	}
	if day11 != 2 {
		t.Errorf("day 11 events = %d, want 2", day11)
	}
}
