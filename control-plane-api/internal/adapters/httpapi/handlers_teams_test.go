package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nuevo-idp/control-plane-api/internal/adapters/memoryrepo"
	"github.com/nuevo-idp/control-plane-api/internal/application"
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

	logger := zap.NewNop()
	return NewServer(services, logger), teamRepo, appRepo, envRepo, appEnvRepo, secretRepo, secretBindingRepo, codeRepo, depRepo, gitopsRepo
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
