package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDeclareCodeRepositoryEndpoint_CreatesRepo(t *testing.T) {
	server, _, appRepo, _, _, _, _, codeRepo, _, _ := newTestServer()
	mux := server.Routes()
	ctx := httptest.NewRequest("", "/", nil).Context()

	if err := server.services.CreateTeam(ctx, "team-1", "Platform", "test"); err != nil {
		t.Fatalf("CreateTeam failed: %v", err)
	}
	if err := server.services.CreateApplication(ctx, "app-1", "App", "team-1", "test"); err != nil {
		t.Fatalf("CreateApplication failed: %v", err)
	}
	if _, err := appRepo.GetByID(ctx, "app-1"); err != nil {
		t.Fatalf("expected app to exist, got error: %v", err)
	}

	body, _ := json.Marshal(map[string]string{
		"id":            "repo-1",
		"applicationId": "app-1",
	})

	req := httptest.NewRequest(http.MethodPost, "/commands/code-repositories", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected %d, got %d", http.StatusCreated, rec.Code)
	}

	repo, err := codeRepo.GetByID(ctx, "repo-1")
	if err != nil || repo == nil {
		t.Fatalf("expected code repo to be created, got err=%v repo=%v", err, repo)
	}
}

func TestDeclareDeploymentRepositoryEndpoint_CreatesRepo(t *testing.T) {
	server, _, appRepo, _, _, _, _, _, depRepo, _ := newTestServer()
	mux := server.Routes()
	ctx := httptest.NewRequest("", "/", nil).Context()

	if err := server.services.CreateTeam(ctx, "team-1", "Platform", "test"); err != nil {
		t.Fatalf("CreateTeam failed: %v", err)
	}
	if err := server.services.CreateApplication(ctx, "app-1", "App", "team-1", "test"); err != nil {
		t.Fatalf("CreateApplication failed: %v", err)
	}
	if _, err := appRepo.GetByID(ctx, "app-1"); err != nil {
		t.Fatalf("expected app to exist, got error: %v", err)
	}

	body, _ := json.Marshal(map[string]string{
		"id":              "dep-1",
		"applicationId":   "app-1",
		"deploymentModel": "GitOpsPerApplication",
	})

	req := httptest.NewRequest(http.MethodPost, "/commands/deployment-repositories", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected %d, got %d", http.StatusCreated, rec.Code)
	}

	repo, err := depRepo.GetByID(ctx, "dep-1")
	if err != nil || repo == nil {
		t.Fatalf("expected dep repo to be created, got err=%v repo=%v", err, repo)
	}
}

func TestDeclareGitOpsIntegrationEndpoint_CreatesIntegration(t *testing.T) {
	server, _, appRepo, _, _, _, _, _, depRepo, gitopsRepo := newTestServer()
	mux := server.Routes()
	ctx := httptest.NewRequest("", "/", nil).Context()

	if err := server.services.CreateTeam(ctx, "team-1", "Platform", "test"); err != nil {
		t.Fatalf("CreateTeam failed: %v", err)
	}
	if err := server.services.CreateApplication(ctx, "app-1", "App", "team-1", "test"); err != nil {
		t.Fatalf("CreateApplication failed: %v", err)
	}
	if _, err := appRepo.GetByID(ctx, "app-1"); err != nil {
		t.Fatalf("expected app to exist, got error: %v", err)
	}

	// Necesitamos un deployment repo asociado a app-1
	if err := server.services.DeclareDeploymentRepository(ctx, "dep-1", "app-1", "GitOpsPerApplication", "test"); err != nil {
		t.Fatalf("DeclareDeploymentRepository failed: %v", err)
	}
	if _, err := depRepo.GetByID(ctx, "dep-1"); err != nil {
		t.Fatalf("expected dep repo to exist, got error: %v", err)
	}

	body, _ := json.Marshal(map[string]string{
		"id":                     "gi-1",
		"applicationId":          "app-1",
		"deploymentRepositoryId": "dep-1",
	})

	req := httptest.NewRequest(http.MethodPost, "/commands/gitops-integrations", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected %d, got %d", http.StatusCreated, rec.Code)
	}

	gi, err := gitopsRepo.GetByID(ctx, "gi-1")
	if err != nil || gi == nil {
			 t.Fatalf("expected gitops integration to be created, got err=%v gi=%v", err, gi)
	}
}

func TestDeclareCodeRepositoryEndpoint_FailsWhenApplicationNotFound(t *testing.T) {
	server, _, _, _, _, _, _, _, _, _ := newTestServer()
	mux := server.Routes()

	body, _ := json.Marshal(map[string]string{
		"id":            "repo-1",
		"applicationId": "does-not-exist",
	})

	req := httptest.NewRequest(http.MethodPost, "/commands/code-repositories", bytes.NewReader(body))
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

func TestDeclareCodeRepositoryEndpoint_FailsWhenDuplicateID(t *testing.T) {
	server, _, appRepo, _, _, _, _, codeRepo, _, _ := newTestServer()
	mux := server.Routes()
	ctx := httptest.NewRequest("", "/", nil).Context()

	if err := server.services.CreateTeam(ctx, "team-1", "Platform", "test"); err != nil {
		t.Fatalf("CreateTeam failed: %v", err)
	}
	if err := server.services.CreateApplication(ctx, "app-1", "App", "team-1", "test"); err != nil {
		t.Fatalf("CreateApplication failed: %v", err)
	}
	if _, err := appRepo.GetByID(ctx, "app-1"); err != nil {
		t.Fatalf("expected app to exist, got error: %v", err)
	}

	if err := server.services.DeclareCodeRepository(ctx, "repo-1", "app-1", "test"); err != nil {
		t.Fatalf("DeclareCodeRepository via service failed: %v", err)
	}
	if _, err := codeRepo.GetByID(ctx, "repo-1"); err != nil {
		t.Fatalf("expected repo to exist, got error: %v", err)
	}

	body, _ := json.Marshal(map[string]string{
		"id":            "repo-1",
		"applicationId": "app-1",
	})

	req := httptest.NewRequest(http.MethodPost, "/commands/code-repositories", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected %d when declaring duplicate code repo, got %d", http.StatusConflict, rec.Code)
	}
	var errPayload map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &errPayload); err != nil {
		t.Fatalf("expected JSON error payload, got %v", err)
	}
	if errPayload["code"] != "code_repository_already_exists" {
		t.Fatalf("expected error code 'code_repository_already_exists', got %q", errPayload["code"])
	}
}

func TestDeclareDeploymentRepositoryEndpoint_FailsWhenApplicationNotFound(t *testing.T) {
	server, _, _, _, _, _, _, _, _, _ := newTestServer()
	mux := server.Routes()

	body, _ := json.Marshal(map[string]string{
		"id":            "dep-1",
		"applicationId": "does-not-exist",
		"deploymentModel": "GitOpsPerApplication",
	})

	req := httptest.NewRequest(http.MethodPost, "/commands/deployment-repositories", bytes.NewReader(body))
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

func TestDeclareDeploymentRepositoryEndpoint_FailsWhenDuplicateID(t *testing.T) {
	server, _, appRepo, _, _, _, _, _, depRepo, _ := newTestServer()
	mux := server.Routes()
	ctx := httptest.NewRequest("", "/", nil).Context()

	if err := server.services.CreateTeam(ctx, "team-1", "Platform", "test"); err != nil {
		t.Fatalf("CreateTeam failed: %v", err)
	}
	if err := server.services.CreateApplication(ctx, "app-1", "App", "team-1", "test"); err != nil {
		t.Fatalf("CreateApplication failed: %v", err)
	}
	if _, err := appRepo.GetByID(ctx, "app-1"); err != nil {
		t.Fatalf("expected app to exist, got error: %v", err)
	}

	if err := server.services.DeclareDeploymentRepository(ctx, "dep-1", "app-1", "GitOpsPerApplication", "test"); err != nil {
		t.Fatalf("DeclareDeploymentRepository via service failed: %v", err)
	}
	if _, err := depRepo.GetByID(ctx, "dep-1"); err != nil {
		t.Fatalf("expected dep repo to exist, got error: %v", err)
	}

	body, _ := json.Marshal(map[string]string{
		"id":              "dep-1",
		"applicationId":   "app-1",
		"deploymentModel": "GitOpsPerApplication",
	})

	req := httptest.NewRequest(http.MethodPost, "/commands/deployment-repositories", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected %d when declaring duplicate deployment repo, got %d", http.StatusConflict, rec.Code)
	}
	var errPayload map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &errPayload); err != nil {
		t.Fatalf("expected JSON error payload, got %v", err)
	}
	if errPayload["code"] != "deployment_repository_already_exists" {
		t.Fatalf("expected error code 'deployment_repository_already_exists', got %q", errPayload["code"])
	}
}

func TestDeclareGitOpsIntegrationEndpoint_FailsWhenApplicationNotFound(t *testing.T) {
	server, _, _, _, _, _, _, _, _, _ := newTestServer()
	mux := server.Routes()

	body, _ := json.Marshal(map[string]string{
		"id":                     "gi-1",
		"applicationId":          "does-not-exist",
		"deploymentRepositoryId": "dep-1",
	})

	req := httptest.NewRequest(http.MethodPost, "/commands/gitops-integrations", bytes.NewReader(body))
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

func TestDeclareGitOpsIntegrationEndpoint_FailsWhenDeploymentRepositoryNotFound(t *testing.T) {
	server, _, appRepo, _, _, _, _, _, _, _ := newTestServer()
	mux := server.Routes()
	ctx := httptest.NewRequest("", "/", nil).Context()

	if err := server.services.CreateTeam(ctx, "team-1", "Platform", "test"); err != nil {
		t.Fatalf("CreateTeam failed: %v", err)
	}
	if err := server.services.CreateApplication(ctx, "app-1", "App", "team-1", "test"); err != nil {
		t.Fatalf("CreateApplication failed: %v", err)
	}
	if _, err := appRepo.GetByID(ctx, "app-1"); err != nil {
		t.Fatalf("expected app to exist, got error: %v", err)
	}

	body, _ := json.Marshal(map[string]string{
		"id":                     "gi-1",
		"applicationId":          "app-1",
		"deploymentRepositoryId": "does-not-exist",
	})

	req := httptest.NewRequest(http.MethodPost, "/commands/gitops-integrations", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected %d when deployment repo is missing, got %d", http.StatusNotFound, rec.Code)
	}
	var errPayload map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &errPayload); err != nil {
		t.Fatalf("expected JSON error payload, got %v", err)
	}
	if errPayload["code"] != "deployment_repository_not_found" {
		t.Fatalf("expected error code 'deployment_repository_not_found', got %q", errPayload["code"])
	}
}

func TestDeclareGitOpsIntegrationEndpoint_FailsWhenDeploymentRepositoryBelongsToDifferentApplication(t *testing.T) {
	server, _, appRepo, _, _, _, _, _, depRepo, _ := newTestServer()
	mux := server.Routes()
	ctx := httptest.NewRequest("", "/", nil).Context()

	// Creamos dos aplicaciones
	if err := server.services.CreateTeam(ctx, "team-1", "Platform", "test"); err != nil {
		t.Fatalf("CreateTeam failed: %v", err)
	}
	if err := server.services.CreateApplication(ctx, "app-1", "App1", "team-1", "test"); err != nil {
		t.Fatalf("CreateApplication app-1 failed: %v", err)
	}
	if err := server.services.CreateApplication(ctx, "app-2", "App2", "team-1", "test"); err != nil {
		t.Fatalf("CreateApplication app-2 failed: %v", err)
	}
	if _, err := appRepo.GetByID(ctx, "app-1"); err != nil {
		t.Fatalf("expected app-1 to exist, got error: %v", err)
	}
	if _, err := appRepo.GetByID(ctx, "app-2"); err != nil {
		t.Fatalf("expected app-2 to exist, got error: %v", err)
	}

	// Declaramos un deployment repo asociado a app-2
	if err := server.services.DeclareDeploymentRepository(ctx, "dep-1", "app-2", "GitOpsPerApplication", "test"); err != nil {
		t.Fatalf("DeclareDeploymentRepository failed: %v", err)
	}
	if _, err := depRepo.GetByID(ctx, "dep-1"); err != nil {
		t.Fatalf("expected dep repo to exist, got error: %v", err)
	}

	body, _ := json.Marshal(map[string]string{
		"id":                     "gi-1",
		"applicationId":          "app-1",
		"deploymentRepositoryId": "dep-1",
	})

	req := httptest.NewRequest(http.MethodPost, "/commands/gitops-integrations", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected %d when deployment repo belongs to different application, got %d", http.StatusBadRequest, rec.Code)
	}
	var errPayload map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &errPayload); err != nil {
		t.Fatalf("expected JSON error payload, got %v", err)
	}
	if errPayload["code"] != "deployment_repository_wrong_application" {
		t.Fatalf("expected error code 'deployment_repository_wrong_application', got %q", errPayload["code"])
	}
}

func TestDeclareGitOpsIntegrationEndpoint_FailsWhenDuplicateID(t *testing.T) {
	server, _, appRepo, _, _, _, _, _, depRepo, gitopsRepo := newTestServer()
	mux := server.Routes()
	ctx := httptest.NewRequest("", "/", nil).Context()

	if err := server.services.CreateTeam(ctx, "team-1", "Platform", "test"); err != nil {
		t.Fatalf("CreateTeam failed: %v", err)
	}
	if err := server.services.CreateApplication(ctx, "app-1", "App", "team-1", "test"); err != nil {
		t.Fatalf("CreateApplication failed: %v", err)
	}
	if _, err := appRepo.GetByID(ctx, "app-1"); err != nil {
		t.Fatalf("expected app to exist, got error: %v", err)
	}

	if err := server.services.DeclareDeploymentRepository(ctx, "dep-1", "app-1", "GitOpsPerApplication", "test"); err != nil {
		t.Fatalf("DeclareDeploymentRepository failed: %v", err)
	}
	if _, err := depRepo.GetByID(ctx, "dep-1"); err != nil {
		t.Fatalf("expected dep repo to exist, got error: %v", err)
	}

	// Creamos una integración inicial vía servicio
	if err := server.services.DeclareGitOpsIntegration(ctx, "gi-1", "app-1", "dep-1", "test"); err != nil {
		t.Fatalf("DeclareGitOpsIntegration via service failed: %v", err)
	}
	if _, err := gitopsRepo.GetByID(ctx, "gi-1"); err != nil {
		t.Fatalf("expected gitops integration to exist, got error: %v", err)
	}

	body, _ := json.Marshal(map[string]string{
		"id":                     "gi-1",
		"applicationId":          "app-1",
		"deploymentRepositoryId": "dep-1",
	})

	req := httptest.NewRequest(http.MethodPost, "/commands/gitops-integrations", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("expected %d when declaring duplicate gitops integration, got %d", http.StatusConflict, rec.Code)
	}
	var errPayload map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &errPayload); err != nil {
		t.Fatalf("expected JSON error payload, got %v", err)
	}
	if errPayload["code"] != "gitops_integration_already_exists" {
		t.Fatalf("expected error code 'gitops_integration_already_exists', got %q", errPayload["code"])
	}
}

