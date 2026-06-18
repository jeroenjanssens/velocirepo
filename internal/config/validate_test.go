package config

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestValidateSourceGitHubOK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/org/repo" {
			w.WriteHeader(404)
			return
		}
		w.WriteHeader(200)
	}))
	defer srv.Close()

	// Override by using a custom client that redirects to our test server
	client := srv.Client()
	transport := &rewriteTransport{base: http.DefaultTransport, target: srv.URL}
	client.Transport = transport

	result := ValidateSource(context.Background(), ValidationOptions{
		Client:  client,
		Timeout: 5 * time.Second,
	}, "github", "org/repo")

	if !result.OK {
		t.Errorf("expected OK, got error: %s", result.Error)
	}
}

func TestValidateSourceGitHub404(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	}))
	defer srv.Close()

	client := srv.Client()
	client.Transport = &rewriteTransport{base: http.DefaultTransport, target: srv.URL}

	result := ValidateSource(context.Background(), ValidationOptions{
		Client:  client,
		Timeout: 5 * time.Second,
	}, "github", "org/nonexistent")

	if result.OK {
		t.Error("expected failure for 404")
	}
	if result.StatusCode != 404 {
		t.Errorf("StatusCode = %d, want 404", result.StatusCode)
	}
}

func TestValidateSourcePyPIOK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer srv.Close()

	client := srv.Client()
	client.Transport = &rewriteTransport{base: http.DefaultTransport, target: srv.URL}

	result := ValidateSource(context.Background(), ValidationOptions{
		Client: client,
	}, "pypi", "requests")

	if !result.OK {
		t.Errorf("expected OK, got error: %s", result.Error)
	}
}

func TestValidateSourceTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(200)
	}))
	defer srv.Close()

	client := &http.Client{
		Timeout:   50 * time.Millisecond,
		Transport: &rewriteTransport{base: http.DefaultTransport, target: srv.URL},
	}

	result := ValidateSource(context.Background(), ValidationOptions{
		Client: client,
	}, "pypi", "slow-pkg")

	if result.OK {
		t.Error("expected failure due to timeout")
	}
}

func TestValidateSourceCancellation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(200)
	}))
	defer srv.Close()

	client := srv.Client()
	client.Transport = &rewriteTransport{base: http.DefaultTransport, target: srv.URL}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result := ValidateSource(ctx, ValidationOptions{Client: client}, "github", "org/repo")
	if result.OK {
		t.Error("expected failure due to cancelled context")
	}
}

func TestValidateProject(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/repos/org/repo" {
			w.WriteHeader(200)
		} else if r.URL.Path == "/pypi/mypkg/json" {
			w.WriteHeader(200)
		} else {
			w.WriteHeader(404)
		}
	}))
	defer srv.Close()

	client := srv.Client()
	client.Transport = &rewriteTransport{base: http.DefaultTransport, target: srv.URL}

	results := ValidateProject(context.Background(), ValidationOptions{Client: client}, "test", Project{
		GitHub: StringList{"org/repo"},
		PyPI:         StringList{"mypkg"},
		CRAN:         StringList{"nonexistent"},
	})

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	if !results[0].OK {
		t.Error("github should be OK")
	}
	if !results[1].OK {
		t.Error("pypi should be OK")
	}
	if results[2].OK {
		t.Error("cran should fail")
	}
}

// rewriteTransport rewrites request URLs to point at the test server
type rewriteTransport struct {
	base   http.RoundTripper
	target string
}

func (t *rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.URL.Scheme = "http"
	req.URL.Host = t.target[len("http://"):]
	return t.base.RoundTrip(req)
}
