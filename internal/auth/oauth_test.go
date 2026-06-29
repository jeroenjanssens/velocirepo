package auth

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestRunOnReadyCallback(t *testing.T) {
	flow := &OAuthFlow{
		AuthURL:      "https://example.com/authorize",
		TokenURL:     "https://example.com/token",
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		RedirectURI:  "http://127.0.0.1:0/callback",
		Scopes:       []string{"read", "write"},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	urlCh := make(chan string, 1)
	go func() {
		flow.Run(ctx, func(authURL string) {
			urlCh <- authURL
		})
	}()

	var gotURL string
	select {
	case gotURL = <-urlCh:
	case <-time.After(2 * time.Second):
		t.Fatal("onReady was not called")
	}
	cancel()

	u, err := url.Parse(gotURL)
	if err != nil {
		t.Fatalf("invalid auth URL: %v", err)
	}
	if u.Query().Get("state") == "" {
		t.Error("auth URL missing state parameter")
	}
	if u.Query().Get("client_id") != "test-client" {
		t.Errorf("client_id = %q, want %q", u.Query().Get("client_id"), "test-client")
	}
	if u.Query().Get("scope") != "read write" {
		t.Errorf("scope = %q, want %q", u.Query().Get("scope"), "read write")
	}
}

func TestRunStateMismatch(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close()

	flow := &OAuthFlow{
		AuthURL:      "https://example.com/authorize",
		TokenURL:     "https://example.com/token",
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		RedirectURI:  fmt.Sprintf("http://127.0.0.1:%d/callback", port),
		Scopes:       []string{"read"},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		_, err := flow.Run(ctx, func(authURL string) {
			callbackURL := fmt.Sprintf("http://127.0.0.1:%d/callback?state=wrong&code=abc", port)
			resp, err := http.Get(callbackURL)
			if err == nil {
				resp.Body.Close()
			}
		})
		errCh <- err
	}()

	select {
	case err := <-errCh:
		if err == nil {
			t.Fatal("expected error for state mismatch, got nil")
		}
		if !strings.Contains(err.Error(), "state mismatch") {
			t.Errorf("unexpected error: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("test timed out")
	}
}
