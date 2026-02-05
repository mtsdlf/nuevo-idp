//go:build ignore

package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nuevo-idp/control-plane-api/internal/adapters/memoryrepo"
	"github.com/nuevo-idp/control-plane-api/internal/application"
	"github.com/nuevo-idp/control-plane-api/internal/domain"
	"go.uber.org/zap"
)

func newTestServer() (*Server, *memoryrepo.TeamRepository, *memoryrepo.ApplicationRepository, *memoryrepo.EnvironmentRepository, *memoryrepo.ApplicationEnvironmentRepository, *memoryrepo.SecretRepository, *memoryrepo.SecretBindingRepository, *memoryrepo.CodeRepositoryRepository, *memoryrepo.DeploymentRepositoryRepository, *memoryrepo.GitOpsIntegrationRepository) {
	teamRepo := memoryrepo.NewTeamRepository()
	appRepo := memoryrepo.NewApplicationRepository()
	envRepo := memoryrepo.NewEnvironmentRepository()
	appEnvRepo := memoryrepo.NewApplicationEnvironmentRepository()
	secretRepo := memoryrepo.NewSecretRepository()
	secretBindingRepo := memoryrepo.NewSecretBindingRepository()
	codeRepo := memoryrepo.NewCodeRepositoryRepository()
	depRepo := memoryrepo.NewDeploymentRepositoryRepository()
	gitopsRepo := memoryrepo.NewGitOpsIntegrationRepository()

	services := &application.Services{
		Teams:                   teamRepo,
		Applications:            appRepo,
		Environments:            envRepo,
		ApplicationEnvironments: appEnvRepo,
		Secrets:                 secretRepo,
		SecretBindings:          secretBindingRepo,
		CodeRepositories:        codeRepo,
		DeploymentRepositories:  depRepo,
		GitOpsIntegrations:      gitopsRepo,
	}

	return NewServer(services, zap.NewNop()), teamRepo, appRepo, envRepo, appEnvRepo, secretRepo, secretBindingRepo, codeRepo, depRepo, gitopsRepo
}

func TestCreateTeamEndpoint_CreatesTeam(t *testing.T) {
	server, teamRepo, _, _, _, _, _, _, _, _ := newTestServer()
	mux := server.Routes()

	body, _ := json.Marshal(map[string]string{
		"id":   "team-1",
		"name": "Platform",
	})

	req := httptest.NewRequest(http.MethodPost, "/commands/teams", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected %d, got %d", http.StatusCreated, rec.Code)
	}

	team, err := teamRepo.GetByID(req.Context(), "team-1")
	if err != nil || team == nil {
		t.Fatalf("expected team to be created, got err=%v team=%v", err, team)
	}
}

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

	if rec.Code == http.StatusCreated {
		t.Fatalf("expected failure because secret is not Active, got %d", rec.Code)
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

func TestStartSecretRotationEndpoint_TransitionsToRotating(t *testing.T) {
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

func TestCompleteSecretRotationEndpoint_TransitionsToActive(t *testing.T) {
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

func TestGetApplicationQuery_ReturnsApplication(t *testing.T) {
	server, _, appRepo, _, _, _, _, _, _, _ := newTestServer()
	mux := server.Routes()
	ctx := httptest.NewRequest("", "/", nil).Context()

	if err := server.services.CreateTeam(ctx, "team-1", "Platform", "test"); err != nil {
		f.Fatalf("CreateTeam failed: %v", err)
	}
	if err := server.services.CreateApplication(ctx, "app-1", "App", "team-1", "test"); err != nil {
		f.Fatalf("CreateApplication failed: %v", err)
	}
	if _, err := appRepo.GetByID(ctx, "app-1"); err != nil {
		f.Fatalf("expected app to exist, got error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/queries/applications?id=app-1", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		f.Fatalf("expected %d, got %d", http.StatusOK, rec.Code)
	}
}
