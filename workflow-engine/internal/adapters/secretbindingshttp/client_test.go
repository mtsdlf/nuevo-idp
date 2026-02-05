package secretbindingshttp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestUpdateSecretBindingsForSecret_Success(t *testing.T) {
	var gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	c := NewClient(server.URL)
	if err := c.UpdateSecretBindingsForSecret(context.Background(), "sec-1"); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if gotPath != "/secrets/bindings/update" {
		t.Fatalf("expected path /secrets/bindings/update, got %s", gotPath)
	}
}

func TestUpdateSecretBindingsForSecret_SendsInternalAuthHeader(t *testing.T) {
	_ = os.Setenv("INTERNAL_AUTH_TOKEN", "test-token")
	t.Cleanup(func() { _ = os.Unsetenv("INTERNAL_AUTH_TOKEN") })

	var gotHeader string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeader = r.Header.Get("X-Internal-Token")
		w.WriteHeader(http.StatusAccepted)
	}))
	t.Cleanup(server.Close)

	c := NewClient(server.URL)
	if err := c.UpdateSecretBindingsForSecret(context.Background(), "sec-1"); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if gotHeader != "test-token" {
		t.Fatalf("expected X-Internal-Token header to be 'test-token', got %q", gotHeader)
	}
}

func TestUpdateSecretBindingsForSecret_Non2xxReturnsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte("bad gateway"))
	}))
	defer server.Close()

	c := NewClient(server.URL)
	err := c.UpdateSecretBindingsForSecret(context.Background(), "sec-1")
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if e, ok := err.(*Error); !ok {
		t.Fatalf("expected *Error, got %T", err)
	} else if e.Status != http.StatusBadGateway {
		t.Fatalf("expected status %d, got %d", http.StatusBadGateway, e.Status)
	}
}
