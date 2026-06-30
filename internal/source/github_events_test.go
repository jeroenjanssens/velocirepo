package source

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func graphqlHandler(responses map[string]string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req struct {
			Query     string                 `json:"query"`
			Variables map[string]interface{} `json:"variables"`
		}
		json.Unmarshal(body, &req)

		for key, resp := range responses {
			if contains(req.Query, key) {
				w.Write([]byte(resp))
				return
			}
		}
		w.Write([]byte(`{"data":{}}`))
	})
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func TestGitHubEventsFetchStargazers(t *testing.T) {
	resp := `{"data":{"repository":{"stargazers":{"edges":[
		{"starredAt":"2025-06-10T10:00:00Z","node":{"login":"alice"}},
		{"starredAt":"2025-06-10T11:00:00Z","node":{"login":"bob"}}
	],"pageInfo":{"hasNextPage":false,"endCursor":"c1"}}}}}`

	srv := httptest.NewServer(graphqlHandler(map[string]string{
		"stargazers":   resp,
		"forks":        `{"data":{"repository":{"forks":{"nodes":[],"pageInfo":{"hasNextPage":false,"endCursor":""}}}}}`,
		"issues":       `{"data":{"repository":{"issues":{"nodes":[],"pageInfo":{"hasNextPage":false,"endCursor":""}}}}}`,
		"pullRequests": `{"data":{"repository":{"pullRequests":{"nodes":[],"pageInfo":{"hasNextPage":false,"endCursor":""}}}}}`,
	}))
	defer srv.Close()

	g := &GitHubEvents{
		Client:  srv.Client(),
		Token:   "test-token",
		Repo:    "owner/repo",
		BaseURL: srv.URL,
	}

	events, err := g.FetchEvents(context.Background(), juneFetchOptions("my-project", 10, 10))
	if err != nil {
		t.Fatalf("FetchEvents failed: %v", err)
	}

	if len(events) != 2 {
		t.Fatalf("got %d events, want 2", len(events))
	}
	if events[0].Type != "star" || events[0].Tags["user"] != "alice" {
		t.Errorf("events[0] = %+v, want star/alice", events[0])
	}
	if events[1].Type != "star" || events[1].Tags["user"] != "bob" {
		t.Errorf("events[1] = %+v, want star/bob", events[1])
	}
}

func TestGitHubEventsFetchAllTypes(t *testing.T) {
	responses := map[string]string{
		"stargazers": `{"data":{"repository":{"stargazers":{"edges":[
			{"starredAt":"2025-06-10T10:00:00Z","node":{"login":"alice"}}
		],"pageInfo":{"hasNextPage":false,"endCursor":""}}}}}`,
		"forks": `{"data":{"repository":{"forks":{"nodes":[
			{"createdAt":"2025-06-10T11:00:00Z","owner":{"login":"bob"}}
		],"pageInfo":{"hasNextPage":false,"endCursor":""}}}}}`,
		"issues": `{"data":{"repository":{"issues":{"nodes":[
			{"createdAt":"2025-06-10T12:00:00Z","closedAt":"2025-06-10T14:00:00Z","author":{"login":"carol"}}
		],"pageInfo":{"hasNextPage":false,"endCursor":""}}}}}`,
		"pullRequests": `{"data":{"repository":{"pullRequests":{"nodes":[
			{"createdAt":"2025-06-10T13:00:00Z","closedAt":"2025-06-10T15:00:00Z","mergedAt":"2025-06-10T15:00:00Z","author":{"login":"dave"}}
		],"pageInfo":{"hasNextPage":false,"endCursor":""}}}}}`,
	}

	srv := httptest.NewServer(graphqlHandler(responses))
	defer srv.Close()

	g := &GitHubEvents{
		Client:  srv.Client(),
		Token:   "test-token",
		Repo:    "owner/repo",
		BaseURL: srv.URL,
	}

	events, err := g.FetchEvents(context.Background(), juneFetchOptions("my-project", 10, 10))
	if err != nil {
		t.Fatalf("FetchEvents failed: %v", err)
	}

	expected := []struct {
		eventType string
		user      string
	}{
		{"star", "alice"},
		{"fork", "bob"},
		{"issue_open", "carol"},
		{"issue_close", "carol"},
		{"pr_open", "dave"},
		{"pr_merge", "dave"},
	}

	if len(events) != len(expected) {
		t.Fatalf("got %d events, want %d", len(events), len(expected))
	}

	for i, want := range expected {
		if events[i].Type != want.eventType {
			t.Errorf("events[%d].Type = %q, want %q", i, events[i].Type, want.eventType)
		}
		if events[i].Tags["user"] != want.user {
			t.Errorf("events[%d].User = %q, want %q", i, events[i].Tags["user"], want.user)
		}
		if events[i].ProjectID != "my-project" {
			t.Errorf("events[%d].ProjectID = %q, want %q", i, events[i].ProjectID, "my-project")
		}
		if events[i].Target != "owner/repo" {
			t.Errorf("events[%d].GitHubRepo = %q, want %q", i, events[i].Target, "owner/repo")
		}
	}
}

func TestGitHubEventsDateFiltering(t *testing.T) {
	responses := map[string]string{
		"stargazers": `{"data":{"repository":{"stargazers":{"edges":[
			{"starredAt":"2025-06-12T10:00:00Z","node":{"login":"alice"}},
			{"starredAt":"2025-06-10T10:00:00Z","node":{"login":"bob"}},
			{"starredAt":"2025-06-08T10:00:00Z","node":{"login":"carol"}}
		],"pageInfo":{"hasNextPage":false,"endCursor":""}}}}}`,
		"forks":        `{"data":{"repository":{"forks":{"nodes":[],"pageInfo":{"hasNextPage":false,"endCursor":""}}}}}`,
		"issues":       `{"data":{"repository":{"issues":{"nodes":[],"pageInfo":{"hasNextPage":false,"endCursor":""}}}}}`,
		"pullRequests": `{"data":{"repository":{"pullRequests":{"nodes":[],"pageInfo":{"hasNextPage":false,"endCursor":""}}}}}`,
	}

	srv := httptest.NewServer(graphqlHandler(responses))
	defer srv.Close()

	g := &GitHubEvents{
		Client:  srv.Client(),
		Repo:    "owner/repo",
		BaseURL: srv.URL,
	}

	events, err := g.FetchEvents(context.Background(), juneFetchOptions("test", 9, 11))
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

func TestGitHubEventsPagination(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req struct {
			Query     string                 `json:"query"`
			Variables map[string]interface{} `json:"variables"`
		}
		json.Unmarshal(body, &req)

		if !contains(req.Query, "stargazers") {
			if contains(req.Query, "forks") {
				w.Write([]byte(`{"data":{"repository":{"forks":{"nodes":[],"pageInfo":{"hasNextPage":false,"endCursor":""}}}}}`))
			} else if contains(req.Query, "issues") {
				w.Write([]byte(`{"data":{"repository":{"issues":{"nodes":[],"pageInfo":{"hasNextPage":false,"endCursor":""}}}}}`))
			} else {
				w.Write([]byte(`{"data":{"repository":{"pullRequests":{"nodes":[],"pageInfo":{"hasNextPage":false,"endCursor":""}}}}}`))
			}
			return
		}

		callCount++
		if callCount == 1 {
			w.Write([]byte(`{"data":{"repository":{"stargazers":{"edges":[
				{"starredAt":"2025-06-10T12:00:00Z","node":{"login":"alice"}}
			],"pageInfo":{"hasNextPage":true,"endCursor":"cursor1"}}}}}`))
		} else {
			w.Write([]byte(`{"data":{"repository":{"stargazers":{"edges":[
				{"starredAt":"2025-06-10T11:00:00Z","node":{"login":"bob"}}
			],"pageInfo":{"hasNextPage":false,"endCursor":"cursor2"}}}}}`))
		}
	}))
	defer srv.Close()

	g := &GitHubEvents{
		Client:  srv.Client(),
		Repo:    "owner/repo",
		BaseURL: srv.URL,
	}

	events, err := g.FetchEvents(context.Background(), juneFetchOptions("test", 10, 10))
	if err != nil {
		t.Fatalf("FetchEvents failed: %v", err)
	}

	if len(events) != 2 {
		t.Fatalf("got %d events, want 2", len(events))
	}
	if events[0].Tags["user"] != "alice" || events[1].Tags["user"] != "bob" {
		t.Errorf("unexpected users: %s, %s", events[0].Tags["user"], events[1].Tags["user"])
	}
}

func TestGitHubEventsInvalidRepo(t *testing.T) {
	g := &GitHubEvents{
		Client: http.DefaultClient,
		Repo:   "invalid",
	}

	_, err := g.FetchEvents(context.Background(), fetchOptions("test", time.Now(), time.Now()))
	if err == nil {
		t.Fatal("expected error for invalid repo")
	}
}

func TestGitHubEventsAuthHeader(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertBearerToken(t, r, "my-secret-token")
		w.Write([]byte(`{"data":{"repository":{"stargazers":{"edges":[],"pageInfo":{"hasNextPage":false,"endCursor":""}},"forks":{"nodes":[],"pageInfo":{"hasNextPage":false,"endCursor":""}},"issues":{"nodes":[],"pageInfo":{"hasNextPage":false,"endCursor":""}},"pullRequests":{"nodes":[],"pageInfo":{"hasNextPage":false,"endCursor":""}}}}}`))
	}))
	defer srv.Close()

	g := &GitHubEvents{
		Client:  srv.Client(),
		Token:   "my-secret-token",
		Repo:    "owner/repo",
		BaseURL: srv.URL,
	}

	g.FetchEvents(context.Background(), fetchOptions("test", time.Now().AddDate(0, 0, -7), time.Now()))
}

func TestGitHubEventsPRNotMerged(t *testing.T) {
	responses := map[string]string{
		"stargazers": `{"data":{"repository":{"stargazers":{"edges":[],"pageInfo":{"hasNextPage":false,"endCursor":""}}}}}`,
		"forks":      `{"data":{"repository":{"forks":{"nodes":[],"pageInfo":{"hasNextPage":false,"endCursor":""}}}}}`,
		"issues":     `{"data":{"repository":{"issues":{"nodes":[],"pageInfo":{"hasNextPage":false,"endCursor":""}}}}}`,
		"pullRequests": `{"data":{"repository":{"pullRequests":{"nodes":[
			{"createdAt":"2025-06-10T10:00:00Z","closedAt":"2025-06-10T12:00:00Z","mergedAt":null,"author":{"login":"alice"}}
		],"pageInfo":{"hasNextPage":false,"endCursor":""}}}}}`,
	}

	srv := httptest.NewServer(graphqlHandler(responses))
	defer srv.Close()

	g := &GitHubEvents{
		Client:  srv.Client(),
		Repo:    "owner/repo",
		BaseURL: srv.URL,
	}

	events, err := g.FetchEvents(context.Background(), juneFetchOptions("test", 10, 10))
	if err != nil {
		t.Fatalf("FetchEvents failed: %v", err)
	}

	if len(events) != 1 {
		t.Fatalf("got %d events, want 1 (only pr_open, no merge)", len(events))
	}
	if events[0].Type != "pr_open" {
		t.Errorf("EventType = %q, want pr_open", events[0].Type)
	}
}

func TestGitHubEventsIssueCloseOutOfRange(t *testing.T) {
	responses := map[string]string{
		"stargazers": `{"data":{"repository":{"stargazers":{"edges":[],"pageInfo":{"hasNextPage":false,"endCursor":""}}}}}`,
		"forks":      `{"data":{"repository":{"forks":{"nodes":[],"pageInfo":{"hasNextPage":false,"endCursor":""}}}}}`,
		"issues": `{"data":{"repository":{"issues":{"nodes":[
			{"createdAt":"2025-06-10T10:00:00Z","closedAt":"2025-06-20T10:00:00Z","author":{"login":"alice"}}
		],"pageInfo":{"hasNextPage":false,"endCursor":""}}}}}`,
		"pullRequests": `{"data":{"repository":{"pullRequests":{"nodes":[],"pageInfo":{"hasNextPage":false,"endCursor":""}}}}}`,
	}

	srv := httptest.NewServer(graphqlHandler(responses))
	defer srv.Close()

	g := &GitHubEvents{
		Client:  srv.Client(),
		Repo:    "owner/repo",
		BaseURL: srv.URL,
	}

	events, err := g.FetchEvents(context.Background(), juneFetchOptions("test", 10, 11))
	if err != nil {
		t.Fatalf("FetchEvents failed: %v", err)
	}

	if len(events) != 1 {
		t.Fatalf("got %d events, want 1 (issue_open only, close is out of range)", len(events))
	}
	if events[0].Type != "issue_open" {
		t.Errorf("EventType = %q, want issue_open", events[0].Type)
	}
}

func TestGitHubEventsGraphQLError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"errors":[{"message":"Bad credentials"}]}`))
	}))
	defer srv.Close()

	g := &GitHubEvents{
		Client:  srv.Client(),
		Repo:    "owner/repo",
		BaseURL: srv.URL,
	}

	_, err := g.FetchEvents(context.Background(), fetchOptions("test", time.Now(), time.Now()))
	if err == nil {
		t.Fatal("expected error for GraphQL error response")
	}
}
