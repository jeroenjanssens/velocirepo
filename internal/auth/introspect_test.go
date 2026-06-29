package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestIntrospectWithURL(t *testing.T) {
	t.Run("parses active token response", func(t *testing.T) {
		expiresAt := time.Now().Add(30 * 24 * time.Hour).Unix()
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "POST" {
				t.Errorf("expected POST, got %s", r.Method)
			}
			if ct := r.Header.Get("Content-Type"); ct != "application/x-www-form-urlencoded" {
				t.Errorf("expected form content-type, got %s", ct)
			}
			body, _ := io.ReadAll(r.Body)
			if !bytes.Contains(body, []byte("token=test-token")) {
				t.Errorf("expected token in body, got %s", body)
			}
			if !bytes.Contains(body, []byte("client_id=my-client")) {
				t.Errorf("expected client_id in body, got %s", body)
			}
			if !bytes.Contains(body, []byte("client_secret=my-secret")) {
				t.Errorf("expected client_secret in body, got %s", body)
			}
			json.NewEncoder(w).Encode(IntrospectResult{
				Active:    true,
				ExpiresAt: expiresAt,
				Scope:     "r_organization_social",
			})
		}))
		defer srv.Close()

		ctx := context.Background()
		result, err := introspectWithURL(ctx, srv.URL, "test-token", "my-client", "my-secret")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !result.Active {
			t.Error("expected active=true")
		}
		if result.ExpiresAt != expiresAt {
			t.Errorf("expected expires_at=%d, got %d", expiresAt, result.ExpiresAt)
		}
		if result.Scope != "r_organization_social" {
			t.Errorf("expected scope=r_organization_social, got %s", result.Scope)
		}
	})

	t.Run("inactive token", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(IntrospectResult{Active: false})
		}))
		defer srv.Close()

		ctx := context.Background()
		result, err := introspectWithURL(ctx, srv.URL, "token", "client", "secret")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Active {
			t.Error("expected active=false")
		}
	})

	t.Run("server error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, "internal error")
		}))
		defer srv.Close()

		ctx := context.Background()
		_, err := introspectWithURL(ctx, srv.URL, "token", "client", "secret")
		if err == nil {
			t.Fatal("expected error for 500 response")
		}
	})
}

func TestCheckLinkedInTokenExpiry(t *testing.T) {
	t.Run("skips when credentials missing", func(t *testing.T) {
		var buf bytes.Buffer
		ctx := context.Background()
		checkLinkedInTokenExpiryWith(ctx, "http://unused", "token", "", "", &buf)
		if buf.Len() != 0 {
			t.Errorf("expected no output when credentials missing, got %q", buf.String())
		}
	})

	t.Run("warns when token inactive", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(IntrospectResult{Active: false})
		}))
		defer srv.Close()

		var buf bytes.Buffer
		ctx := context.Background()
		checkLinkedInTokenExpiryWith(ctx, srv.URL, "token", "client", "secret", &buf)
		if !strings.Contains(buf.String(), "no longer active") {
			t.Errorf("expected inactive warning, got %q", buf.String())
		}
	})

	t.Run("warns when token expires soon", func(t *testing.T) {
		expiresAt := time.Now().Add(3 * 24 * time.Hour).Unix()
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(IntrospectResult{Active: true, ExpiresAt: expiresAt})
		}))
		defer srv.Close()

		var buf bytes.Buffer
		ctx := context.Background()
		checkLinkedInTokenExpiryWith(ctx, srv.URL, "token", "client", "secret", &buf)
		if !strings.Contains(buf.String(), "expires in") {
			t.Errorf("expected expiry warning, got %q", buf.String())
		}
	})

	t.Run("warns when token expires today", func(t *testing.T) {
		expiresAt := time.Now().Add(1 * time.Hour).Unix()
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(IntrospectResult{Active: true, ExpiresAt: expiresAt})
		}))
		defer srv.Close()

		var buf bytes.Buffer
		ctx := context.Background()
		checkLinkedInTokenExpiryWith(ctx, srv.URL, "token", "client", "secret", &buf)
		if !strings.Contains(buf.String(), "expires today") {
			t.Errorf("expected 'expires today' warning, got %q", buf.String())
		}
	})

	t.Run("no warning when expiry far away", func(t *testing.T) {
		expiresAt := time.Now().Add(30 * 24 * time.Hour).Unix()
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(IntrospectResult{Active: true, ExpiresAt: expiresAt})
		}))
		defer srv.Close()

		var buf bytes.Buffer
		ctx := context.Background()
		checkLinkedInTokenExpiryWith(ctx, srv.URL, "token", "client", "secret", &buf)
		if buf.Len() != 0 {
			t.Errorf("expected no output for far-away expiry, got %q", buf.String())
		}
	})
}
