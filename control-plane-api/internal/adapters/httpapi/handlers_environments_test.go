package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nuevo-idp/control-plane-api/internal/domain"
)

func TestCreateEnvironmentEndpoint_CreatesEnvironment(t *testing.T) {
	server, _, _, envRepo, _, _, _, _, _, _ := newTestServer()
	mux := server.Routes()

	body, _ := json.Marshal(map[string]string{
		"id":   "env-dev",
		"name": "Dev",
	})

	req := httptest.NewRequest(http.MethodPost, "/commands/environments", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected %d, got %d", http.StatusCreated, rec.Code)
	}

	env, err := envRepo.GetByID(req.Context(), "env-dev")
	if err != nil || env == nil {
		t.Fatalf("expected environment to be created, got err=%v env=%v", err, env)
	}
}

func TestDeclareApplicationEnvironmentEndpoint_UsesDomainRules(t *testing.T) {
	server, _, appRepo, envRepo, appEnvRepo, _, _, _, _, _ := newTestServer()
	mux := server.Routes()
	ctx := httptest.NewRequest("", "/", nil).Context()

	// Creamos aplicación y entorno vía servicios para respetar reglas previas
	if err := server.services.CreateTeam(ctx, "team-1", "Platform", "test"); err != nil {
		t.Fatalf("CreateTeam failed: %v", err)
	}
	if err := server.services.CreateApplication(ctx, "app-1", "App", "team-1", "test"); err != nil {
		t.Fatalf("CreateApplication failed: %v", err)
	}
	if err := server.services.CreateEnvironment(ctx, "env-dev", "Dev", "test"); err != nil {
		t.Fatalf("CreateEnvironment failed: %v", err)
	}

	if _, err := appRepo.GetByID(ctx, "app-1"); err != nil {
		t.Fatalf("expected app to exist, got error: %v", err)
	}
	if _, err := envRepo.GetByID(ctx, "env-dev"); err != nil {
		t.Fatalf("expected env to exist, got error: %v", err)
	}

	body, _ := json.Marshal(map[string]string{
		"id":            "ae-1",
		"applicationId": "app-1",
		"environmentId": "env-dev",
	})

	req := httptest.NewRequest(http.MethodPost, "/commands/application-environments", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected %d, got %d", http.StatusCreated, rec.Code)
	}

	ae, err := appEnvRepo.GetByID(req.Context(), "ae-1")
	if err != nil || ae == nil {
		t.Fatalf("expected application environment to be created, got err=%v ae=%v", err, ae)
	}
}

func TestCompleteApplicationEnvironmentProvisioningEndpoint_ActivatesState(t *testing.T) {
	server, _, appRepo, envRepo, appEnvRepo, _, _, _, _, _ := newTestServer()
	mux := server.Routes()
	ctx := httptest.NewRequest("", "/", nil).Context()

	// Preparamos app env en estado Declared usando los servicios
	if err := server.services.CreateTeam(ctx, "team-1", "Platform", "test"); err != nil {
		t.Fatalf("CreateTeam failed: %v", err)
	}
	if err := server.services.CreateApplication(ctx, "app-1", "App", "team-1", "test"); err != nil {
		t.Fatalf("CreateApplication failed: %v", err)
	}
	if err := server.services.CreateEnvironment(ctx, "env-dev", "Dev", "test"); err != nil {
		t.Fatalf("CreateEnvironment failed: %v", err)
	}
	if _, err := appRepo.GetByID(ctx, "app-1"); err != nil {
		t.Fatalf("expected app to exist, got error: %v", err)
	}
	if _, err := envRepo.GetByID(ctx, "env-dev"); err != nil {
		t.Fatalf("expected env to exist, got error: %v", err)
	}
	if err := server.services.DeclareApplicationEnvironment(ctx, "ae-1", "app-1", "env-dev", "test"); err != nil {
		t.Fatalf("DeclareApplicationEnvironment failed: %v", err)
	}

	body, _ := json.Marshal(map[string]string{
		"id": "ae-1",
	})

	req := httptest.NewRequest(http.MethodPost, "/commands/application-environments/complete-provisioning", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected %d, got %d", http.StatusAccepted, rec.Code)
	}

	updated, err := appEnvRepo.GetByID(req.Context(), "ae-1")
	if err != nil || updated == nil {
		t.Fatalf("expected application environment, got err=%v ae=%v", err, updated)
	}
	if updated.State != domain.ApplicationEnvironmentStateActive {
		t.Fatalf("expected state %q, got %q", domain.ApplicationEnvironmentStateActive, updated.State)
	}
}
