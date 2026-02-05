package gitproviderhttp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestCreateRepository_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/github/repos" {
			t.Fatalf("expected path /github/repos, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	c := NewClient(server.URL)
	if err := c.CreateRepository(context.Background(), "owner", "name", true); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestCreateRepository_Non2xxReturnsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte("bad gateway"))
	}))
	defer server.Close()

	c := NewClient(server.URL)
	err := c.CreateRepository(context.Background(), "owner", "name", true)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	if e, ok := err.(*Error); !ok {
		t.Fatalf("expected *Error, got %T", err)
	} else if e.Status != http.StatusBadGateway {
		t.Fatalf("expected status %d, got %d", http.StatusBadGateway, e.Status)
	}
}

func TestCreateRepository_SendsInternalAuthHeader(t *testing.T) {
	_ = os.Setenv("INTERNAL_AUTH_TOKEN", "test-token")
	t.Cleanup(func() { _ = os.Unsetenv("INTERNAL_AUTH_TOKEN") })

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-Internal-Token"); got != "test-token" {
			t.Fatalf("expected X-Internal-Token header to be 'test-token', got %q", got)
		}
		w.WriteHeader(http.StatusCreated)
	}))
	t.Cleanup(server.Close)

	c := NewClient(server.URL)
	if err := c.CreateRepository(context.Background(), "owner", "name", true); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}
