package harbor

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRotateRobotToken_Success(t *testing.T) {
	var gotMethod, gotUser, gotPass string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		user, pass, _ := r.BasicAuth()
		gotUser, gotPass = user, pass
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"token":"robot-token-123"}`))
	}))
	t.Cleanup(server.Close)

	c := &Client{
		baseURL:    server.URL,
		username:   "robot",
		password:   "secret",
		httpClient: &http.Client{Timeout: 2 * time.Second},
	}

	ctx := context.Background()
	token, err := c.RotateRobotToken(ctx)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if token != "robot-token-123" {
		t.Fatalf("expected token robot-token-123, got %s", token)
	}

	if gotMethod != http.MethodPost {
		t.Fatalf("expected POST, got %s", gotMethod)
	}
	if gotUser != "robot" || gotPass != "secret" {
		t.Fatalf("expected basic auth robot/secret, got %s/%s", gotUser, gotPass)
	}
}

func TestRotateRobotToken_Non2xxReturnsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(server.Close)

	c := &Client{
		baseURL:    server.URL,
		username:   "robot",
		password:   "secret",
		httpClient: &http.Client{Timeout: 2 * time.Second},
	}

	ctx := context.Background()
	if _, err := c.RotateRobotToken(ctx); err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestRotateRobotToken_InvalidJSONReturnsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("not-json"))
	}))
	t.Cleanup(server.Close)

	c := &Client{
		baseURL:    server.URL,
		username:   "robot",
		password:   "secret",
		httpClient: &http.Client{Timeout: 2 * time.Second},
	}

	ctx := context.Background()
	if _, err := c.RotateRobotToken(ctx); err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestRotateRobotToken_MissingTokenFieldReturnsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"not_token":"x"}`))
	}))
	t.Cleanup(server.Close)

	c := &Client{
		baseURL:    server.URL,
		username:   "robot",
		password:   "secret",
		httpClient: &http.Client{Timeout: 2 * time.Second},
	}

	ctx := context.Background()
	if _, err := c.RotateRobotToken(ctx); err == nil {
		t.Fatalf("expected error, got nil")
	}
}
