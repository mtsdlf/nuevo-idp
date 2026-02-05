package appenvprovhttp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestPost_Success(t *testing.T) {
	var receivedPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	c := NewClient(server.URL)
	if err := c.ApplyBranchProtection(context.Background(), "ae-1"); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if receivedPath != "/appenv/branch-protection" {
		t.Fatalf("expected path /appenv/branch-protection, got %s", receivedPath)
	}
}

func TestPost_SendsInternalAuthHeader(t *testing.T) {
	_ = os.Setenv("INTERNAL_AUTH_TOKEN", "test-token")
	t.Cleanup(func() { _ = os.Unsetenv("INTERNAL_AUTH_TOKEN") })

	var gotHeader string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeader = r.Header.Get("X-Internal-Token")
		w.WriteHeader(http.StatusAccepted)
	}))
	t.Cleanup(server.Close)

	c := NewClient(server.URL)
	if err := c.ProvisionSecrets(context.Background(), "ae-1"); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if gotHeader != "test-token" {
		t.Fatalf("expected X-Internal-Token header to be 'test-token', got %q", gotHeader)
	}
}

func TestPost_Non2xxReturnsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("bad request"))
	}))
	defer server.Close()

	c := NewClient(server.URL)
	err := c.ProvisionSecrets(context.Background(), "ae-1")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	if e, ok := err.(*Error); !ok {
		t.Fatalf("expected *Error, got %T", err)
	} else if e.Status != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, e.Status)
	}
}
