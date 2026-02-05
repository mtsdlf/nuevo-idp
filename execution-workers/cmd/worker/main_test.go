package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"go.uber.org/zap"
	"io"
	"strings"
)

func TestHandleCreateGitHubRepo_MethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/github/repos", nil)
	rec := httptest.NewRecorder()

	handleCreateGitHubRepo(zap.NewNop(), rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected %d, got %d", http.StatusMethodNotAllowed, rec.Code)
	}
}

func TestHandleCreateGitHubRepo_InvalidJSON(t *testing.T) {
	body := bytes.NewBufferString("not-json")
	req := httptest.NewRequest(http.MethodPost, "/github/repos", body)
	rec := httptest.NewRecorder()

	handleCreateGitHubRepo(zap.NewNop(), rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestHandleCreateGitHubRepo_MissingFields(t *testing.T) {
	body := bytes.NewBufferString(`{"name":""}`)
	req := httptest.NewRequest(http.MethodPost, "/github/repos", body)
	rec := httptest.NewRecorder()

	handleCreateGitHubRepo(zap.NewNop(), rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestHandleCreateGitHubRepo_MissingToken(t *testing.T) {
	_ = os.Unsetenv("GITHUB_TOKEN")

	body := bytes.NewBufferString(`{"owner":"me","name":"repo","private":true}`)
	req := httptest.NewRequest(http.MethodPost, "/github/repos", body)
	rec := httptest.NewRecorder()

	handleCreateGitHubRepo(zap.NewNop(), rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

func TestHandleAppEnvHandlers_MethodNotAllowed(t *testing.T) {
	handlers := []struct {
		name string
		h    http.HandlerFunc
	}{
		{"branch", func(w http.ResponseWriter, r *http.Request) { handleAppEnvBranchProtection(zap.NewNop(), w, r) }},
		{"secrets", func(w http.ResponseWriter, r *http.Request) { handleAppEnvSecrets(zap.NewNop(), w, r) }},
		{"bindings", func(w http.ResponseWriter, r *http.Request) { handleAppEnvSecretBindings(zap.NewNop(), w, r) }},
		{"gitops", func(w http.ResponseWriter, r *http.Request) { handleAppEnvGitOpsVerify(zap.NewNop(), w, r) }},
	}

	for _, tc := range handlers {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		t.Run(tc.name, func(t *testing.T) {
			tc.h(rec, req)

			if rec.Code != http.StatusMethodNotAllowed {
				t.Fatalf("expected %d, got %d", http.StatusMethodNotAllowed, rec.Code)
			}
		})
	}
}

//nolint:dupl
func TestHandleAppEnvHandlers_InvalidJSON(t *testing.T) {
	handlers := []http.HandlerFunc{
		func(w http.ResponseWriter, r *http.Request) { handleAppEnvBranchProtection(zap.NewNop(), w, r) },
		func(w http.ResponseWriter, r *http.Request) { handleAppEnvSecrets(zap.NewNop(), w, r) },
		func(w http.ResponseWriter, r *http.Request) { handleAppEnvSecretBindings(zap.NewNop(), w, r) },
		func(w http.ResponseWriter, r *http.Request) { handleAppEnvGitOpsVerify(zap.NewNop(), w, r) },
	}

	for _, h := range handlers {
		body := bytes.NewBufferString("not-json")
		req := httptest.NewRequest(http.MethodPost, "/", body)
		rec := httptest.NewRecorder()

		h(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected %d, got %d", http.StatusBadRequest, rec.Code)
		}
	}
}

//nolint:dupl
func TestHandleAppEnvHandlers_MissingID(t *testing.T) {
	handlers := []http.HandlerFunc{
		func(w http.ResponseWriter, r *http.Request) { handleAppEnvBranchProtection(zap.NewNop(), w, r) },
		func(w http.ResponseWriter, r *http.Request) { handleAppEnvSecrets(zap.NewNop(), w, r) },
		func(w http.ResponseWriter, r *http.Request) { handleAppEnvSecretBindings(zap.NewNop(), w, r) },
		func(w http.ResponseWriter, r *http.Request) { handleAppEnvGitOpsVerify(zap.NewNop(), w, r) },
	}

	for _, h := range handlers {
		body := bytes.NewBufferString(`{"applicationEnvironmentId":""}`)
		req := httptest.NewRequest(http.MethodPost, "/", body)
		rec := httptest.NewRecorder()

		h(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected %d, got %d", http.StatusBadRequest, rec.Code)
		}
	}
}

//nolint:dupl
func TestHandleAppEnvHandlers_AcceptsValidRequest(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	t.Cleanup(server.Close)

	_ = os.Setenv("APPENV_GITOPS_VERIFY_ENDPOINT", server.URL)
	t.Cleanup(func() { _ = os.Unsetenv("APPENV_GITOPS_VERIFY_ENDPOINT") })

	body := bytes.NewBufferString(`{"applicationEnvironmentId":"ae-1"}`)
	req := httptest.NewRequest(http.MethodPost, "/appenv/gitops-verify", body)
	rec := httptest.NewRecorder()

	handleAppEnvGitOpsVerify(zap.NewNop(), rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected %d, got %d", http.StatusAccepted, rec.Code)
	}
	if !called {
		t.Fatalf("expected endpoint to be called")
	}
}

func TestHandleAppEnvSecretBindings_CallsConfiguredEndpoint(t *testing.T) { //nolint:dupl // patrón de test repetido intencionalmente para distintos endpoints appenv
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	_ = os.Setenv("APPENV_SECRET_BINDINGS_ENDPOINT", server.URL)
	t.Cleanup(func() { _ = os.Unsetenv("APPENV_SECRET_BINDINGS_ENDPOINT") })

	body := bytes.NewBufferString(`{"applicationEnvironmentId":"ae-1"}`)
	req := httptest.NewRequest(http.MethodPost, "/appenv/secret-bindings", body)
	rec := httptest.NewRecorder()

	handleAppEnvSecretBindings(zap.NewNop(), rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected %d, got %d", http.StatusAccepted, rec.Code)
	}
	if !called {
		t.Fatalf("expected endpoint to be called")
	}
}

func TestHandleAppEnvSecrets_CallsConfiguredEndpoint(t *testing.T) { //nolint:dupl // patrón de test repetido intencionalmente para distintos endpoints appenv
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	_ = os.Setenv("APPENV_SECRETS_ENDPOINT", server.URL)
	t.Cleanup(func() { _ = os.Unsetenv("APPENV_SECRETS_ENDPOINT") })

	body := bytes.NewBufferString(`{"applicationEnvironmentId":"ae-1"}`)
	req := httptest.NewRequest(http.MethodPost, "/appenv/secrets", body)
	rec := httptest.NewRecorder()

	handleAppEnvSecrets(zap.NewNop(), rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected %d, got %d", http.StatusAccepted, rec.Code)
	}
	if !called {
		t.Fatalf("expected endpoint to be called")
	}
}

func TestHandleAppEnvGitOpsVerify_CallsConfiguredEndpoint(t *testing.T) { //nolint:dupl // patrón de test repetido intencionalmente para distintos endpoints appenv
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	t.Cleanup(server.Close)

	_ = os.Setenv("APPENV_GITOPS_VERIFY_ENDPOINT", server.URL)
	t.Cleanup(func() { _ = os.Unsetenv("APPENV_GITOPS_VERIFY_ENDPOINT") })

	body := bytes.NewBufferString(`{"applicationEnvironmentId":"ae-1"}`)
	req := httptest.NewRequest(http.MethodPost, "/appenv/gitops-verify", body)
	rec := httptest.NewRecorder()

	handleAppEnvGitOpsVerify(zap.NewNop(), rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected %d, got %d", http.StatusAccepted, rec.Code)
	}
	if !called {
		t.Fatalf("expected endpoint to be called")
	}
}

func TestHandleAppEnvBranchProtection_AppliesProtectionWithGitHub(t *testing.T) {
	// Fake GitHub API endpoint
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(server.Close)

	_ = os.Setenv("GITHUB_TOKEN", "dummy-token")
	_ = os.Setenv("GITHUB_API_URL", server.URL+"/")
	t.Cleanup(func() {
		_ = os.Unsetenv("GITHUB_TOKEN")
		_ = os.Unsetenv("GITHUB_API_URL")
	})

	body := bytes.NewBufferString(`{"applicationEnvironmentId":"ae-1"}`)
	req := httptest.NewRequest(http.MethodPost, "/appenv/branch-protection", body)
	rec := httptest.NewRecorder()

	handleAppEnvBranchProtection(zap.NewNop(), rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected %d, got %d", http.StatusAccepted, rec.Code)
	}
}

func TestHandleSecretBindingsUpdate_RequiresSecretID(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/secrets/bindings/update", bytes.NewBufferString(`{"secretId":""}`))
	rec := httptest.NewRecorder()

	handleSecretBindingsUpdate(zap.NewNop(), rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestHandleSecretBindingsUpdate_HarborNotConfiguredStillAccepted(t *testing.T) {
	_ = os.Unsetenv("HARBOR_URL")
	_ = os.Unsetenv("HARBOR_ROBOT_USERNAME")
	_ = os.Unsetenv("HARBOR_ROBOT_PASSWORD")

	req := httptest.NewRequest(http.MethodPost, "/secrets/bindings/update", bytes.NewBufferString(`{"secretId":"sec-1"}`))
	rec := httptest.NewRecorder()

	handleSecretBindingsUpdate(zap.NewNop(), rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected %d, got %d", http.StatusAccepted, rec.Code)
	}
}

func TestHandleSecretBindingsUpdate_RotatesHarborAndCallsBindingsEndpoint(t *testing.T) {
	// Fake bindings propagation endpoint
	var calledBindings bool
	var bindingsBody string
	bindingsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calledBindings = true
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		b, _ := io.ReadAll(r.Body)
		bindingsBody = string(b)
		w.WriteHeader(http.StatusAccepted)
	}))
	t.Cleanup(bindingsServer.Close)

	// Fake Harbor endpoint that returns a rotated token
	harborServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"token":"robot-token-123"}`))
	}))
	t.Cleanup(harborServer.Close)

	_ = os.Setenv("HARBOR_URL", harborServer.URL)
	_ = os.Setenv("HARBOR_ROBOT_USERNAME", "robot")
	_ = os.Setenv("HARBOR_ROBOT_PASSWORD", "secret")
	_ = os.Setenv("SECRET_BINDINGS_UPDATE_ENDPOINT", bindingsServer.URL)
	t.Cleanup(func() {
		_ = os.Unsetenv("HARBOR_URL")
		_ = os.Unsetenv("HARBOR_ROBOT_USERNAME")
		_ = os.Unsetenv("HARBOR_ROBOT_PASSWORD")
		_ = os.Unsetenv("SECRET_BINDINGS_UPDATE_ENDPOINT")
	})

	req := httptest.NewRequest(http.MethodPost, "/secrets/bindings/update", bytes.NewBufferString(`{"secretId":"sec-1"}`))
	rec := httptest.NewRecorder()

	handleSecretBindingsUpdate(zap.NewNop(), rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected %d, got %d", http.StatusAccepted, rec.Code)
	}
	if !calledBindings {
		t.Fatalf("expected bindings endpoint to be called")
	}
	if !strings.Contains(bindingsBody, "\"secretId\":\"sec-1\"") || !strings.Contains(bindingsBody, "\"token\":\"robot-token-123\"") {
		t.Fatalf("unexpected bindings body: %s", bindingsBody)
	}
}

func TestHandleCreateGitHubRepo_RequiresInternalAuth(t *testing.T) {
	_ = os.Setenv("INTERNAL_AUTH_TOKEN", "test-token")
	t.Cleanup(func() { _ = os.Unsetenv("INTERNAL_AUTH_TOKEN") })

	body := bytes.NewBufferString(`{"owner":"me","name":"repo","private":true}`)
	req := httptest.NewRequest(http.MethodPost, "/github/repos", body)
	rec := httptest.NewRecorder()

	handleCreateGitHubRepo(zap.NewNop(), rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected %d when missing internal auth token, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestHandleSecretBindingsUpdate_RequiresInternalAuth(t *testing.T) {
	_ = os.Setenv("INTERNAL_AUTH_TOKEN", "test-token")
	t.Cleanup(func() { _ = os.Unsetenv("INTERNAL_AUTH_TOKEN") })

	req := httptest.NewRequest(http.MethodPost, "/secrets/bindings/update", bytes.NewBufferString(`{"secretId":"sec-1"}`))
	rec := httptest.NewRecorder()

	handleSecretBindingsUpdate(zap.NewNop(), rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected %d when missing internal auth token, got %d", http.StatusUnauthorized, rec.Code)
	}
}
