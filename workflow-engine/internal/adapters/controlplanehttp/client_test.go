package controlplanehttp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestCompleteApplicationEnvironmentProvisioning_SendsInternalAuthHeader(t *testing.T) {
	_ = os.Setenv("INTERNAL_AUTH_TOKEN", "test-token")
	t.Cleanup(func() { _ = os.Unsetenv("INTERNAL_AUTH_TOKEN") })

	var gotHeader string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeader = r.Header.Get("X-Internal-Token")
		w.WriteHeader(http.StatusAccepted)
	}))
	t.Cleanup(server.Close)

	c := NewClient(server.URL)
	if err := c.CompleteApplicationEnvironmentProvisioning(context.Background(), "ae-1"); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if gotHeader != "test-token" {
		t.Fatalf("expected X-Internal-Token header to be 'test-token', got %q", gotHeader)
	}
}
