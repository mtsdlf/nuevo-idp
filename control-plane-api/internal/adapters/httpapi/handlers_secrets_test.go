package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nuevo-idp/control-plane-api/internal/domain"
)

func TestCreateSecretEndpoint_RequiresOwnerTeamAndStartsDeclared(t *testing.T) {
	server, _, _, _, _, secretRepo, _, _, _, _ := newTestServer()
	mux := server.Routes()
	ctx := httptest.NewRequest("", "/", nil).Context()

	if err := server.services.CreateTeam(ctx, "team-1", "Platform", "test"); err != nil {
		t.Fatalf("CreateTeam failed: %v", err)
	}

	body, _ := json.Marshal(map[string]string{
		"id":          "sec-1",
		"ownerTeamId": "team-1",
		"purpose":     "runtime",
		"sensitivity": "high",
	})

	req := httptest.NewRequest(http.MethodPost, "/commands/secrets", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected %d, got %d", http.StatusCreated, rec.Code)
	}

	sec, err := secretRepo.GetByID(req.Context(), "sec-1")
	if err != nil || sec == nil {
		t.Fatalf("expected secret to be created, got err=%v sec=%v", err, sec)
	}
	if sec.State != domain.SecretStateDeclared {
		t.Fatalf("expected state %q, got %q", domain.SecretStateDeclared, sec.State)
	}
}

func TestDeclareSecretBindingEndpoint_RequiresActiveSecret(t *testing.T) {
	server, _, _, _, _, secretRepo, bindingRepo, _, _, _ := newTestServer()
	mux := server.Routes()
	ctx := httptest.NewRequest("", "/", nil).Context()

	// Creamos team + secret (quedará en Declared)
	if err := server.services.CreateTeam(ctx, "team-1", "Platform", "test"); err != nil {
		t.Fatalf("CreateTeam failed: %v", err)
	}
	if err := server.services.CreateSecret(ctx, "sec-1", "team-1", "runtime", "high", "test"); err != nil {
		t.Fatalf("CreateSecret failed: %v", err)
	}

	body, _ := json.Marshal(map[string]string{
		"id":         "bind-1",
		"secretId":   "sec-1",
		"targetId":   "target-1",
		"targetType": "CodeRepository",
	})

	req := httptest.NewRequest(http.MethodPost, "/commands/secret-bindings", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected %d because secret is not Active, got %d", http.StatusBadRequest, rec.Code)
	}
	var errPayload map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &errPayload); err != nil {
		t.Fatalf("expected JSON error payload, got %v", err)
	}
	if errPayload["code"] != "binding_requires_active_secret" {
		t.Fatalf("expected error code 'binding_requires_active_secret', got %q", errPayload["code"])
	}

	// Activamos el secret y reintentamos
	sec, err := secretRepo.GetByID(ctx, "sec-1")
	if err != nil || sec == nil {
		t.Fatalf("expected secret to exist, got err=%v sec=%v", err, sec)
	}
	sec.State = domain.SecretStateActive
	if err := secretRepo.Save(ctx, sec); err != nil {
		t.Fatalf("saving updated secret failed: %v", err)
	}

	req2 := httptest.NewRequest(http.MethodPost, "/commands/secret-bindings", bytes.NewReader(body))
	rec2 := httptest.NewRecorder()
	mux.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusCreated {
		t.Fatalf("expected %d after activating secret, got %d", http.StatusCreated, rec2.Code)
	}

	bind, err := bindingRepo.GetByID(ctx, "bind-1")
	if err != nil || bind == nil {
		t.Fatalf("expected binding to be created, got err=%v bind=%v", err, bind)
	}
}

func TestStartSecretRotationEndpoint_TransitionsToRotating(t *testing.T) { //nolint:dupl // patrón de test repetido a propósito para cubrir transición feliz y estados inválidos
	server, _, _, _, _, secretRepo, _, _, _, _ := newTestServer()
	mux := server.Routes()
	ctx := httptest.NewRequest("", "/", nil).Context()

	if err := server.services.CreateTeam(ctx, "team-1", "Platform", "test"); err != nil {
		t.Fatalf("CreateTeam failed: %v", err)
	}
	if err := server.services.CreateSecret(ctx, "sec-1", "team-1", "runtime", "high", "test"); err != nil {
		t.Fatalf("CreateSecret failed: %v", err)
	}
	sec, err := secretRepo.GetByID(ctx, "sec-1")
	if err != nil || sec == nil {
		t.Fatalf("expected secret, got err=%v sec=%v", err, sec)
	}
	sec.State = domain.SecretStateActive
	if err := secretRepo.Save(ctx, sec); err != nil {
		t.Fatalf("saving secret failed: %v", err)
	}

	body, _ := json.Marshal(map[string]string{
		"id": "sec-1",
	})
	req := httptest.NewRequest(http.MethodPost, "/commands/secrets/start-rotation", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected %d, got %d", http.StatusAccepted, rec.Code)
	}

	updated, err := secretRepo.GetByID(ctx, "sec-1")
	if err != nil || updated == nil {
		t.Fatalf("expected secret, got err=%v sec=%v", err, updated)
	}
	if updated.State != domain.SecretStateRotating {
		t.Fatalf("expected state %q, got %q", domain.SecretStateRotating, updated.State)
	}
}

func TestStartSecretRotationEndpoint_FailsIfNotActive(t *testing.T) { //nolint:dupl // patrón de test repetido a propósito para cubrir transición feliz y estados inválidos
	server, _, _, _, _, secretRepo, _, _, _, _ := newTestServer()
	mux := server.Routes()
	ctx := httptest.NewRequest("", "/", nil).Context()

	if err := server.services.CreateTeam(ctx, "team-1", "Platform", "test"); err != nil {
		t.Fatalf("CreateTeam failed: %v", err)
	}
	if err := server.services.CreateSecret(ctx, "sec-1", "team-1", "runtime", "high", "test"); err != nil {
		t.Fatalf("CreateSecret failed: %v", err)
	}

	sec, err := secretRepo.GetByID(ctx, "sec-1")
	if err != nil || sec == nil {
		t.Fatalf("expected secret, got err=%v sec=%v", err, sec)
	}
	if sec.State != domain.SecretStateDeclared {
		t.Fatalf("expected initial state %q, got %q", domain.SecretStateDeclared, sec.State)
	}

	body, _ := json.Marshal(map[string]string{
		"id": "sec-1",
	})
	req := httptest.NewRequest(http.MethodPost, "/commands/secrets/start-rotation", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected %d when starting rotation from non-Active state, got %d", http.StatusBadRequest, rec.Code)
	}
	var errPayload map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &errPayload); err != nil {
		t.Fatalf("expected JSON error payload, got %v", err)
	}
	if errPayload["code"] != "secret_invalid_state_for_start_rotation" {
		t.Fatalf("expected error code 'secret_invalid_state_for_start_rotation', got %q", errPayload["code"])
	}
}

func TestCompleteSecretRotationEndpoint_TransitionsToActive(t *testing.T) { //nolint:dupl // patrón de test repetido a propósito para cubrir transición feliz y estados inválidos
	server, _, _, _, _, secretRepo, _, _, _, _ := newTestServer()
	mux := server.Routes()
	ctx := httptest.NewRequest("", "/", nil).Context()

	if err := server.services.CreateTeam(ctx, "team-1", "Platform", "test"); err != nil {
		t.Fatalf("CreateTeam failed: %v", err)
	}
	if err := server.services.CreateSecret(ctx, "sec-1", "team-1", "runtime", "high", "test"); err != nil {
		t.Fatalf("CreateSecret failed: %v", err)
	}

	// Ponemos el secreto en Rotating
	sec, err := secretRepo.GetByID(ctx, "sec-1")
	if err != nil || sec == nil {
		t.Fatalf("expected secret, got err=%v sec=%v", err, sec)
	}
	sec.State = domain.SecretStateRotating
	if err := secretRepo.Save(ctx, sec); err != nil {
		t.Fatalf("saving secret failed: %v", err)
	}

	body, _ := json.Marshal(map[string]string{
		"id": "sec-1",
	})
	req := httptest.NewRequest(http.MethodPost, "/commands/secrets/complete-rotation", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected %d, got %d", http.StatusAccepted, rec.Code)
	}

	updated, err := secretRepo.GetByID(ctx, "sec-1")
	if err != nil || updated == nil {
		t.Fatalf("expected secret, got err=%v sec=%v", err, updated)
	}
	if updated.State != domain.SecretStateActive {
		t.Fatalf("expected state %q, got %q", domain.SecretStateActive, updated.State)
	}
}

func TestCompleteSecretRotationEndpoint_FailsIfNotRotating(t *testing.T) { //nolint:dupl // patrón de test repetido a propósito para cubrir transición feliz y estados inválidos
	server, _, _, _, _, secretRepo, _, _, _, _ := newTestServer()
	mux := server.Routes()
	ctx := httptest.NewRequest("", "/", nil).Context()

	if err := server.services.CreateTeam(ctx, "team-1", "Platform", "test"); err != nil {
		t.Fatalf("CreateTeam failed: %v", err)
	}
	if err := server.services.CreateSecret(ctx, "sec-1", "team-1", "runtime", "high", "test"); err != nil {
		t.Fatalf("CreateSecret failed: %v", err)
	}

	sec, err := secretRepo.GetByID(ctx, "sec-1")
	if err != nil || sec == nil {
		t.Fatalf("expected secret, got err=%v sec=%v", err, sec)
	}
	if sec.State != domain.SecretStateDeclared {
		t.Fatalf("expected initial state %q, got %q", domain.SecretStateDeclared, sec.State)
	}

	body, _ := json.Marshal(map[string]string{
		"id": "sec-1",
	})
	req := httptest.NewRequest(http.MethodPost, "/commands/secrets/complete-rotation", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected %d when completing rotation from non-Rotating state, got %d", http.StatusBadRequest, rec.Code)
	}
	var errPayload map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &errPayload); err != nil {
		t.Fatalf("expected JSON error payload, got %v", err)
	}
	if errPayload["code"] != "secret_invalid_state_for_complete_rotation" {
		t.Fatalf("expected error code 'secret_invalid_state_for_complete_rotation', got %q", errPayload["code"])
	}
}

func TestCompleteSecretRotationEndpoint_FailsIfNotFound(t *testing.T) {
	server, _, _, _, _, _, _, _, _, _ := newTestServer()
	mux := server.Routes()

	body, _ := json.Marshal(map[string]string{
		"id": "does-not-exist",
	})
	req := httptest.NewRequest(http.MethodPost, "/commands/secrets/complete-rotation", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected %d when completing rotation for missing secret, got %d", http.StatusNotFound, rec.Code)
	}
	var errPayload map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &errPayload); err != nil {
		t.Fatalf("expected JSON error payload, got %v", err)
	}
	if errPayload["code"] != "secret_not_found" {
		t.Fatalf("expected error code 'secret_not_found', got %q", errPayload["code"])
	}
}
