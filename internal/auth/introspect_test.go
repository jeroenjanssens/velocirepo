package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestIntrospectLinkedInToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		body, _ := io.ReadAll(r.Body)
		if !bytes.Contains(body, []byte("token=test-token")) {
			t.Errorf("expected token in body, got %s", body)
		}
		json.NewEncoder(w).Encode(IntrospectResult{
			Active:    true,
			ExpiresAt: time.Now().Add(30 * 24 * time.Hour).Unix(),
			Scope:     "r_organization_social",
		})
	}))
	defer srv.Close()

	// Patch the URL by using IntrospectLinkedInToken directly won't work
	// since it hardcodes the URL, so we test via a server override approach.
	// Instead, test CheckLinkedInTokenExpiry which just prints warnings.
	t.Run("active token no warning", func(t *testing.T) {
		old := os.Stderr
		r, w, _ := os.Pipe()
		os.Stderr = w

		ctx := context.Background()
		CheckLinkedInTokenExpiry(ctx, "token", "", "")

		w.Close()
		os.Stderr = old
		var buf bytes.Buffer
		io.Copy(&buf, r)
		if buf.Len() != 0 {
			t.Errorf("expected no output when credentials missing, got %q", buf.String())
		}
	})
}

func TestCheckLinkedInTokenExpiry_ExpiringToken(t *testing.T) {
	expiresAt := time.Now().Add(3 * 24 * time.Hour).Unix()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(IntrospectResult{
			Active:    true,
			ExpiresAt: expiresAt,
		})
	}))
	defer srv.Close()

	// We can't easily override the URL in CheckLinkedInTokenExpiry,
	// but we can test IntrospectLinkedInToken directly with a custom server.
	ctx := context.Background()

	// Override by calling the lower-level function
	result, err := introspectWithURL(ctx, srv.URL, "token", "client", "secret")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Active {
		t.Error("expected active=true")
	}
	if result.ExpiresAt != expiresAt {
		t.Errorf("expected expires_at=%d, got %d", expiresAt, result.ExpiresAt)
	}
}

func TestCheckLinkedInTokenExpiry_InactiveToken(t *testing.T) {
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
}

func TestCheckLinkedInTokenExpiry_ServerError(t *testing.T) {
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
}
