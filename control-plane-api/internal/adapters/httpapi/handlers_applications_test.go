package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/nuevo-idp/control-plane-api/internal/domain"
)

func TestCreateApplicationEndpoint_RequiresExistingTeam(t *testing.T) {
	server, teamRepo, appRepo, _, _, _, _, _, _, _ := newTestServer()
	mux := server.Routes()

	// Primero creamos el team vía servicio directo para simplificar
	if err := server.services.CreateTeam(httptest.NewRequest("", "/", nil).Context(), "team-1", "Platform", "test"); err != nil {
		t.Fatalf("CreateTeam via service failed: %v", err)
	}

	if _, err := teamRepo.GetByID(httptest.NewRequest("", "/", nil).Context(), "team-1"); err != nil {
		t.Fatalf("expected team to exist, got error: %v", err)
	}

	body, _ := json.Marshal(map[string]string{
		"id":     "app-1",
		"name":   "App",
		"teamId": "team-1",
	})

	req := httptest.NewRequest(http.MethodPost, "/commands/applications", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected %d, got %d", http.StatusCreated, rec.Code)
	}

	app, err := appRepo.GetByID(req.Context(), "app-1")
	if err != nil || app == nil {
		t.Fatalf("expected application to be created, got err=%v app=%v", err, app)
	}
}

func TestCreateApplicationEndpoint_FailsWhenTeamNotFound(t *testing.T) {
	server, _, _, _, _, _, _, _, _, _ := newTestServer()
	mux := server.Routes()

	body, _ := json.Marshal(map[string]string{
		"id":     "app-1",
		"name":   "App",
		"teamId": "does-not-exist",
	})

	req := httptest.NewRequest(http.MethodPost, "/commands/applications", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected %d when team is missing, got %d", http.StatusNotFound, rec.Code)
	}
	var errPayload map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &errPayload); err != nil {
		t.Fatalf("expected JSON error payload, got %v", err)
	}
	if errPayload["code"] != "team_not_found" {
		t.Fatalf("expected error code 'team_not_found', got %q", errPayload["code"])
	}
}

func TestCreateApplicationEndpoint_FailsWhenDuplicateID(t *testing.T) {
	server, _, appRepo, _, _, _, _, _, _, _ := newTestServer()
	mux := server.Routes()
	ctx := httptest.NewRequest("", "/", nil).Context()

	if err := server.services.CreateTeam(ctx, "team-1", "Platform", "test"); err != nil {
		t.Fatalf("CreateTeam failed: %v", err)
	}
	if err := server.services.CreateApplication(ctx, "app-1", "App", "team-1", "test"); err != nil {
		t.Fatalf("CreateApplication via service failed: %v", err)
	}
	if _, err := appRepo.GetByID(ctx, "app-1"); err != nil {
		t.Fatalf("expected application to exist, got error: %v", err)
	}

	body, _ := json.Marshal(map[string]string{
		"id":     "app-1",
		"name":   "App",
		"teamId": "team-1",
	})

	req := httptest.NewRequest(http.MethodPost, "/commands/applications", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected %d when creating duplicate application, got %d", http.StatusConflict, rec.Code)
	}
	var errPayload map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &errPayload); err != nil {
		t.Fatalf("expected JSON error payload, got %v", err)
	}
	if errPayload["code"] != "application_already_exists" {
		t.Fatalf("expected error code 'application_already_exists', got %q", errPayload["code"])
	}
}

func TestApproveApplicationEndpoint_TransitionsToApproved(t *testing.T) {
	server, _, appRepo, _, _, _, _, _, _, _ := newTestServer()
	mux := server.Routes()
	ctx := httptest.NewRequest("", "/", nil).Context()

	if err := server.services.CreateTeam(ctx, "team-1", "Platform", "test"); err != nil {
		t.Fatalf("CreateTeam failed: %v", err)
	}
	if err := server.services.CreateApplication(ctx, "app-1", "App", "team-1", "test"); err != nil {
		t.Fatalf("CreateApplication failed: %v", err)
	}

	body, _ := json.Marshal(map[string]string{
		"id": "app-1",
	})
	req := httptest.NewRequest(http.MethodPost, "/commands/applications/approve", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected %d, got %d", http.StatusAccepted, rec.Code)
	}

	app, err := appRepo.GetByID(ctx, "app-1")
	if err != nil || app == nil {
		t.Fatalf("expected application, got err=%v app=%v", err, app)
	}
	if app.State != domain.ApplicationStateApproved {
		t.Fatalf("expected state %q, got %q", domain.ApplicationStateApproved, app.State)
	}
}

func TestApproveApplicationEndpoint_FailsWhenNotFound(t *testing.T) {
	server, _, _, _, _, _, _, _, _, _ := newTestServer()
	mux := server.Routes()

	body, _ := json.Marshal(map[string]string{
		"id": "does-not-exist",
	})
	req := httptest.NewRequest(http.MethodPost, "/commands/applications/approve", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected %d when application is missing, got %d", http.StatusNotFound, rec.Code)
	}
	var errPayload map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &errPayload); err != nil {
		t.Fatalf("expected JSON error payload, got %v", err)
	}
	if errPayload["code"] != "application_not_found" {
		t.Fatalf("expected error code 'application_not_found', got %q", errPayload["code"])
	}
}

func TestApproveApplicationEndpoint_FailsWhenInvalidState(t *testing.T) {
	server, _, appRepo, _, _, _, _, _, _, _ := newTestServer()
	mux := server.Routes()
	ctx := httptest.NewRequest("", "/", nil).Context()

	if err := server.services.CreateTeam(ctx, "team-1", "Platform", "test"); err != nil {
		t.Fatalf("CreateTeam failed: %v", err)
	}
	if err := server.services.CreateApplication(ctx, "app-1", "App", "team-1", "test"); err != nil {
		t.Fatalf("CreateApplication failed: %v", err)
	}
	app, err := appRepo.GetByID(ctx, "app-1")
	if err != nil || app == nil {
		t.Fatalf("expected application, got err=%v app=%v", err, app)
	}
	app.State = domain.ApplicationStateApproved
	if err := appRepo.Save(ctx, app); err != nil {
		t.Fatalf("saving app failed: %v", err)
	}

	body, _ := json.Marshal(map[string]string{
		"id": "app-1",
	})
	req := httptest.NewRequest(http.MethodPost, "/commands/applications/approve", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected %d when approving from invalid state, got %d", http.StatusBadRequest, rec.Code)
	}
	var errPayload map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &errPayload); err != nil {
		t.Fatalf("expected JSON error payload, got %v", err)
	}
	if errPayload["code"] != "application_invalid_state_for_approval" {
		t.Fatalf("expected error code 'application_invalid_state_for_approval', got %q", errPayload["code"])
	}
}

func TestDeprecateApplicationEndpoint_TransitionsToDeprecated(t *testing.T) {
	server, _, appRepo, _, _, _, _, _, _, _ := newTestServer()
	mux := server.Routes()
	ctx := httptest.NewRequest("", "/", nil).Context()

	if err := server.services.CreateTeam(ctx, "team-1", "Platform", "test"); err != nil {
		t.Fatalf("CreateTeam failed: %v", err)
	}
	if err := server.services.CreateApplication(ctx, "app-1", "App", "team-1", "test"); err != nil {
		t.Fatalf("CreateApplication failed: %v", err)
	}
	app, err := appRepo.GetByID(ctx, "app-1")
	if err != nil || app == nil {
		t.Fatalf("expected application, got err=%v app=%v", err, app)
	}
	app.State = domain.ApplicationStateActive
	if err := appRepo.Save(ctx, app); err != nil {
		t.Fatalf("saving app failed: %v", err)
	}

	body, _ := json.Marshal(map[string]string{
		"id": "app-1",
	})
	req := httptest.NewRequest(http.MethodPost, "/commands/applications/deprecate", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected %d, got %d", http.StatusAccepted, rec.Code)
	}

	updated, err := appRepo.GetByID(ctx, "app-1")
	if err != nil || updated == nil {
		t.Fatalf("expected application, got err=%v app=%v", err, updated)
	}
	if updated.State != domain.ApplicationStateDeprecated {
		t.Fatalf("expected state %q, got %q", domain.ApplicationStateDeprecated, updated.State)
	}
}

func TestDeprecateApplicationEndpoint_FailsWhenNotFound(t *testing.T) {
	server, _, _, _, _, _, _, _, _, _ := newTestServer()
	mux := server.Routes()

	body, _ := json.Marshal(map[string]string{
		"id": "does-not-exist",
	})
	req := httptest.NewRequest(http.MethodPost, "/commands/applications/deprecate", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected %d when application is missing, got %d", http.StatusNotFound, rec.Code)
	}
	var errPayload map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &errPayload); err != nil {
		t.Fatalf("expected JSON error payload, got %v", err)
	}
	if errPayload["code"] != "application_not_found" {
		t.Fatalf("expected error code 'application_not_found', got %q", errPayload["code"])
	}
}

func TestDeprecateApplicationEndpoint_FailsWhenInvalidState(t *testing.T) {
	server, _, appRepo, _, _, _, _, _, _, _ := newTestServer()
	mux := server.Routes()
	ctx := httptest.NewRequest("", "/", nil).Context()

	if err := server.services.CreateTeam(ctx, "team-1", "Platform", "test"); err != nil {
		t.Fatalf("CreateTeam failed: %v", err)
	}
	if err := server.services.CreateApplication(ctx, "app-1", "App", "team-1", "test"); err != nil {
		t.Fatalf("CreateApplication failed: %v", err)
	}
	app, err := appRepo.GetByID(ctx, "app-1")
	if err != nil || app == nil {
		t.Fatalf("expected application, got err=%v app=%v", err, app)
	}
	// Estado no Active
	if app.State != domain.ApplicationStateProposed {
		t.Fatalf("expected initial state %q, got %q", domain.ApplicationStateProposed, app.State)
	}

	body, _ := json.Marshal(map[string]string{
		"id": "app-1",
	})
	req := httptest.NewRequest(http.MethodPost, "/commands/applications/deprecate", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected %d when deprecating from non-Active state, got %d", http.StatusBadRequest, rec.Code)
	}
	var errPayload map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &errPayload); err != nil {
		t.Fatalf("expected JSON error payload, got %v", err)
	}
	if errPayload["code"] != "application_invalid_state_for_deprecation" {
		t.Fatalf("expected error code 'application_invalid_state_for_deprecation', got %q", errPayload["code"])
	}
}

func TestStartApplicationOnboardingEndpoint_TransitionsToOnboarding(t *testing.T) {
	server, _, appRepo, _, _, _, _, _, _, _ := newTestServer()
	mux := server.Routes()
	ctx := httptest.NewRequest("", "/", nil).Context()

	if err := server.services.CreateTeam(ctx, "team-1", "Platform", "test"); err != nil {
		t.Fatalf("CreateTeam failed: %v", err)
	}
	if err := server.services.CreateApplication(ctx, "app-1", "App", "team-1", "test"); err != nil {
		t.Fatalf("CreateApplication failed: %v", err)
	}
	if err := server.services.ApproveApplication(ctx, "app-1", "approver"); err != nil {
		t.Fatalf("ApproveApplication failed: %v", err)
	}

	body, _ := json.Marshal(map[string]string{
		"id": "app-1",
	})
	req := httptest.NewRequest(http.MethodPost, "/commands/applications/start-onboarding", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected %d, got %d", http.StatusAccepted, rec.Code)
	}

	updated, err := appRepo.GetByID(ctx, "app-1")
	if err != nil || updated == nil {
		t.Fatalf("expected application, got err=%v app=%v", err, updated)
	}
	if updated.State != domain.ApplicationStateOnboarding {
		t.Fatalf("expected state %q, got %q", domain.ApplicationStateOnboarding, updated.State)
	}
}

func TestStartApplicationOnboardingEndpoint_FailsWhenNotFound(t *testing.T) {
	server, _, _, _, _, _, _, _, _, _ := newTestServer()
	mux := server.Routes()

	body, _ := json.Marshal(map[string]string{
		"id": "does-not-exist",
	})
	req := httptest.NewRequest(http.MethodPost, "/commands/applications/start-onboarding", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected %d when application is missing, got %d", http.StatusNotFound, rec.Code)
	}
	var errPayload map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &errPayload); err != nil {
		t.Fatalf("expected JSON error payload, got %v", err)
	}
	if errPayload["code"] != "application_not_found" {
		t.Fatalf("expected error code 'application_not_found', got %q", errPayload["code"])
	}
}

func TestStartApplicationOnboardingEndpoint_FailsWhenInvalidState(t *testing.T) {
	server, _, appRepo, _, _, _, _, _, _, _ := newTestServer()
	mux := server.Routes()
	ctx := httptest.NewRequest("", "/", nil).Context()

	if err := server.services.CreateTeam(ctx, "team-1", "Platform", "test"); err != nil {
		t.Fatalf("CreateTeam failed: %v", err)
	}
	if err := server.services.CreateApplication(ctx, "app-1", "App", "team-1", "test"); err != nil {
		t.Fatalf("CreateApplication failed: %v", err)
	}
	// No aprobamos la app, así que queda en Proposed
	app, err := appRepo.GetByID(ctx, "app-1")
	if err != nil || app == nil {
		t.Fatalf("expected application, got err=%v app=%v", err, app)
	}
	if app.State != domain.ApplicationStateProposed {
		t.Fatalf("expected initial state %q, got %q", domain.ApplicationStateProposed, app.State)
	}

	body, _ := json.Marshal(map[string]string{
		"id": "app-1",
	})
	req := httptest.NewRequest(http.MethodPost, "/commands/applications/start-onboarding", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected %d when starting onboarding from non-Approved state, got %d", http.StatusBadRequest, rec.Code)
	}
	var errPayload map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &errPayload); err != nil {
		t.Fatalf("expected JSON error payload, got %v", err)
	}
	if errPayload["code"] != "application_invalid_state_for_onboarding" {
		t.Fatalf("expected error code 'application_invalid_state_for_onboarding', got %q", errPayload["code"])
	}
}

func TestActivateApplicationEndpoint_TransitionsToActive(t *testing.T) {
	server, _, appRepo, _, _, _, _, _, _, _ := newTestServer()
	mux := server.Routes()
	ctx := httptest.NewRequest("", "/", nil).Context()

	if err := server.services.CreateTeam(ctx, "team-1", "Platform", "test"); err != nil {
		t.Fatalf("CreateTeam failed: %v", err)
	}
	if err := server.services.CreateApplication(ctx, "app-1", "App", "team-1", "test"); err != nil {
		t.Fatalf("CreateApplication failed: %v", err)
	}
	if err := server.services.ApproveApplication(ctx, "app-1", "approver"); err != nil {
		t.Fatalf("ApproveApplication failed: %v", err)
	}
	if err := server.services.StartApplicationOnboarding(ctx, "app-1", "user"); err != nil {
		t.Fatalf("StartApplicationOnboarding failed: %v", err)
	}

	body, _ := json.Marshal(map[string]string{
		"id": "app-1",
	})
	req := httptest.NewRequest(http.MethodPost, "/commands/applications/activate", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected %d, got %d", http.StatusAccepted, rec.Code)
	}

	updated, err := appRepo.GetByID(ctx, "app-1")
	if err != nil || updated == nil {
		t.Fatalf("expected application, got err=%v app=%v", err, updated)
	}
	if updated.State != domain.ApplicationStateActive {
		t.Fatalf("expected state %q, got %q", domain.ApplicationStateActive, updated.State)
	}
}

func TestActivateApplicationEndpoint_FailsWhenNotFound(t *testing.T) {
	server, _, _, _, _, _, _, _, _, _ := newTestServer()
	mux := server.Routes()

	body, _ := json.Marshal(map[string]string{
		"id": "does-not-exist",
	})
	req := httptest.NewRequest(http.MethodPost, "/commands/applications/activate", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected %d when application is missing, got %d", http.StatusNotFound, rec.Code)
	}
	var errPayload map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &errPayload); err != nil {
		t.Fatalf("expected JSON error payload, got %v", err)
	}
	if errPayload["code"] != "application_not_found" {
		t.Fatalf("expected error code 'application_not_found', got %q", errPayload["code"])
	}
}

func TestActivateApplicationEndpoint_FailsWhenInvalidState(t *testing.T) {
	server, _, appRepo, _, _, _, _, _, _, _ := newTestServer()
	mux := server.Routes()
	ctx := httptest.NewRequest("", "/", nil).Context()

	if err := server.services.CreateTeam(ctx, "team-1", "Platform", "test"); err != nil {
		t.Fatalf("CreateTeam failed: %v", err)
	}
	if err := server.services.CreateApplication(ctx, "app-1", "App", "team-1", "test"); err != nil {
		t.Fatalf("CreateApplication failed: %v", err)
	}
	// No pasamos por onboarding, así que queda en Proposed
	app, err := appRepo.GetByID(ctx, "app-1")
	if err != nil || app == nil {
		t.Fatalf("expected application, got err=%v app=%v", err, app)
	}
	if app.State != domain.ApplicationStateProposed {
		t.Fatalf("expected initial state %q, got %q", domain.ApplicationStateProposed, app.State)
	}

	body, _ := json.Marshal(map[string]string{
		"id": "app-1",
	})
	req := httptest.NewRequest(http.MethodPost, "/commands/applications/activate", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected %d when activating from non-Onboarding state, got %d", http.StatusBadRequest, rec.Code)
	}
	var errPayload map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &errPayload); err != nil {
		t.Fatalf("expected JSON error payload, got %v", err)
	}
	if errPayload["code"] != "application_invalid_state_for_activation" {
		t.Fatalf("expected error code 'application_invalid_state_for_activation', got %q", errPayload["code"])
	}
}

func TestStartApplicationOnboardingEndpoint_RequiresInternalAuth(t *testing.T) {
	_ = os.Setenv("INTERNAL_AUTH_TOKEN", "test-token")
	t.Cleanup(func() { _ = os.Unsetenv("INTERNAL_AUTH_TOKEN") })

	server, _, _, _, _, _, _, _, _, _ := newTestServer()
	mux := server.Routes()

	body, _ := json.Marshal(map[string]string{
		"id": "app-1",
	})
	req := httptest.NewRequest(http.MethodPost, "/commands/applications/start-onboarding", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected %d when missing internal auth token, got %d", http.StatusUnauthorized, rec.Code)
	}
}
