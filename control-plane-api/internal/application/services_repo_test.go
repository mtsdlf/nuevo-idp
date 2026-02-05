package application

import (
	"context"
	"testing"

	"github.com/nuevo-idp/control-plane-api/internal/adapters/memoryrepo"
	"github.com/nuevo-idp/control-plane-api/internal/domain"
)

func TestDeclareCodeRepository_RequiresApplicationAndStartsDeclared(t *testing.T) {
	teamRepo := memoryrepo.NewTeamRepository()
	appRepo := memoryrepo.NewApplicationRepository()
	codeRepo := memoryrepo.NewCodeRepositoryRepository()

	services := &Services{
		Teams:            teamRepo,
		Applications:     appRepo,
		CodeRepositories: codeRepo,
	}

	ctx := context.Background()
	if err := services.CreateTeam(ctx, "team-1", "Platform", "test"); err != nil {
		t.Fatalf("CreateTeam failed: %v", err)
	}
	if err := services.CreateApplication(ctx, "app-1", "App", "team-1", "test"); err != nil {
		t.Fatalf("CreateApplication failed: %v", err)
	}

	if err := services.DeclareCodeRepository(ctx, "repo-1", "app-1", "test"); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	cr, err := codeRepo.GetByID(ctx, "repo-1")
	if err != nil || cr == nil {
		t.Fatalf("expected code repo to be created, got err=%v cr=%v", err, cr)
	}
	if cr.State != domain.CodeRepositoryStateDeclared {
		t.Fatalf("expected state %q, got %q", domain.CodeRepositoryStateDeclared, cr.State)
	}
}

func TestDeclareDeploymentRepository_RequiresApplicationAndStartsDeclared(t *testing.T) {
	teamRepo := memoryrepo.NewTeamRepository()
	appRepo := memoryrepo.NewApplicationRepository()
	depRepo := memoryrepo.NewDeploymentRepositoryRepository()

	services := &Services{
		Teams:                  teamRepo,
		Applications:           appRepo,
		DeploymentRepositories: depRepo,
	}

	ctx := context.Background()
	if err := services.CreateTeam(ctx, "team-1", "Platform", "test"); err != nil {
		t.Fatalf("CreateTeam failed: %v", err)
	}
	if err := services.CreateApplication(ctx, "app-1", "App", "team-1", "test"); err != nil {
		t.Fatalf("CreateApplication failed: %v", err)
	}

	if err := services.DeclareDeploymentRepository(ctx, "dep-1", "app-1", "GitOpsPerApplication", "test"); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	dr, err := depRepo.GetByID(ctx, "dep-1")
	if err != nil || dr == nil {
		t.Fatalf("expected deployment repo to be created, got err=%v dr=%v", err, dr)
	}
	if dr.State != domain.DeploymentRepositoryStateDeclared {
		t.Fatalf("expected state %q, got %q", domain.DeploymentRepositoryStateDeclared, dr.State)
	}
	if dr.DeploymentModel != "GitOpsPerApplication" {
		t.Fatalf("unexpected deployment model: %q", dr.DeploymentModel)
	}
}

func TestDeclareGitOpsIntegration_RequiresConsistentApplication(t *testing.T) {
	teamRepo := memoryrepo.NewTeamRepository()
	appRepo := memoryrepo.NewApplicationRepository()
	depRepo := memoryrepo.NewDeploymentRepositoryRepository()
	gitopsRepo := memoryrepo.NewGitOpsIntegrationRepository()

	services := &Services{
		Teams:                  teamRepo,
		Applications:           appRepo,
		DeploymentRepositories: depRepo,
		GitOpsIntegrations:     gitopsRepo,
	}

	ctx := context.Background()
	if err := services.CreateTeam(ctx, "team-1", "Platform", "test"); err != nil {
		t.Fatalf("CreateTeam failed: %v", err)
	}
	if err := services.CreateApplication(ctx, "app-1", "App", "team-1", "test"); err != nil {
		t.Fatalf("CreateApplication failed: %v", err)
	}

	// Deployment repo de la misma app
	if err := services.DeclareDeploymentRepository(ctx, "dep-1", "app-1", "GitOpsPerApplication", "test"); err != nil {
		t.Fatalf("DeclareDeploymentRepository failed: %v", err)
	}

	if err := services.DeclareGitOpsIntegration(ctx, "gi-1", "app-1", "dep-1", "test"); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	gi, err := gitopsRepo.GetByID(ctx, "gi-1")
	if err != nil || gi == nil {
		t.Fatalf("expected gitops integration to be created, got err=%v gi=%v", err, gi)
	}
	if gi.ApplicationID != "app-1" || gi.DeploymentRepositoryID != "dep-1" {
		t.Fatalf("unexpected integration data: %+v", gi)
	}
}

func TestDeclareGitOpsIntegration_FailsWhenDeploymentRepoBelongsToAnotherApp(t *testing.T) {
	teamRepo := memoryrepo.NewTeamRepository()
	appRepo := memoryrepo.NewApplicationRepository()
	depRepo := memoryrepo.NewDeploymentRepositoryRepository()
	gitopsRepo := memoryrepo.NewGitOpsIntegrationRepository()

	services := &Services{
		Teams:                  teamRepo,
		Applications:           appRepo,
		DeploymentRepositories: depRepo,
		GitOpsIntegrations:     gitopsRepo,
	}

	ctx := context.Background()
	if err := services.CreateTeam(ctx, "team-1", "Platform", "test"); err != nil {
		t.Fatalf("CreateTeam failed: %v", err)
	}
	if err := services.CreateApplication(ctx, "app-1", "App", "team-1", "test"); err != nil {
		t.Fatalf("CreateApplication failed: %v", err)
	}
	if err := services.CreateApplication(ctx, "app-2", "Other", "team-1", "test"); err != nil {
		t.Fatalf("CreateApplication failed: %v", err)
	}

	// Deployment repo asociado a app-2
	if err := services.DeclareDeploymentRepository(ctx, "dep-1", "app-2", "GitOpsPerApplication", "test"); err != nil {
		t.Fatalf("DeclareDeploymentRepository failed: %v", err)
	}

	if err := services.DeclareGitOpsIntegration(ctx, "gi-1", "app-1", "dep-1", "test"); err == nil {
		t.Fatalf("expected error when deployment repo belongs to another app, got nil")
	}
}
