package application

import (
	"context"
	"testing"

	"github.com/nuevo-idp/control-plane-api/internal/adapters/memoryrepo"
	"github.com/nuevo-idp/control-plane-api/internal/domain"
)

func TestCreateEnvironment_SucceedsOnNewID(t *testing.T) {
	envRepo := memoryrepo.NewEnvironmentRepository()

	services := &Services{
		Environments: envRepo,
	}

	ctx := context.Background()
	if err := services.CreateEnvironment(ctx, "env-dev", "Dev", "test"); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestCreateEnvironment_FailsOnDuplicateID(t *testing.T) {
	envRepo := memoryrepo.NewEnvironmentRepository()

	services := &Services{
		Environments: envRepo,
	}

	ctx := context.Background()
	if err := services.CreateEnvironment(ctx, "env-dev", "Dev", "test"); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if err := services.CreateEnvironment(ctx, "env-dev", "Dev Again", "test"); err == nil {
		t.Fatalf("expected error on duplicate id, got nil")
	}
}

func TestDeclareApplicationEnvironment_EnforcesUniqueness(t *testing.T) {
	teamRepo := memoryrepo.NewTeamRepository()
	appRepo := memoryrepo.NewApplicationRepository()
	envRepo := memoryrepo.NewEnvironmentRepository()
	appEnvRepo := memoryrepo.NewApplicationEnvironmentRepository()

	services := &Services{
		Teams:                  teamRepo,
		Applications:           appRepo,
		Environments:           envRepo,
		ApplicationEnvironments: appEnvRepo,
	}

	ctx := context.Background()

	if err := services.CreateTeam(ctx, "team-1", "Team", "test"); err != nil {
		t.Fatalf("CreateTeam failed: %v", err)
	}
	if err := services.CreateApplication(ctx, "app-1", "App", "team-1", "test"); err != nil {
		t.Fatalf("CreateApplication failed: %v", err)
	}
	if err := services.CreateEnvironment(ctx, "env-dev", "Dev", "test"); err != nil {
		t.Fatalf("CreateEnvironment failed: %v", err)
	}

	if err := services.DeclareApplicationEnvironment(ctx, "ae-1", "app-1", "env-dev", "test"); err != nil {
		t.Fatalf("expected first declaration to succeed, got %v", err)
	}

	if err := services.DeclareApplicationEnvironment(ctx, "ae-2", "app-1", "env-dev", "test"); err == nil {
		t.Fatalf("expected error on duplicate application/environment pair, got nil")
	}
}

func TestCompleteApplicationEnvironmentProvisioning_TransitionsToActive(t *testing.T) {
	teamRepo := memoryrepo.NewTeamRepository()
	appRepo := memoryrepo.NewApplicationRepository()
	envRepo := memoryrepo.NewEnvironmentRepository()
	appEnvRepo := memoryrepo.NewApplicationEnvironmentRepository()

	services := &Services{
		Teams:                  teamRepo,
		Applications:           appRepo,
		Environments:           envRepo,
		ApplicationEnvironments: appEnvRepo,
	}

	ctx := context.Background()

	if err := services.CreateTeam(ctx, "team-1", "Team", "test"); err != nil {
		t.Fatalf("CreateTeam failed: %v", err)
	}
	if err := services.CreateApplication(ctx, "app-1", "App", "team-1", "test"); err != nil {
		t.Fatalf("CreateApplication failed: %v", err)
	}
	if err := services.CreateEnvironment(ctx, "env-dev", "Dev", "test"); err != nil {
		t.Fatalf("CreateEnvironment failed: %v", err)
	}
	if err := services.DeclareApplicationEnvironment(ctx, "ae-1", "app-1", "env-dev", "test"); err != nil {
		t.Fatalf("DeclareApplicationEnvironment failed: %v", err)
	}

	if err := services.CompleteApplicationEnvironmentProvisioning(ctx, "ae-1", "test-workflow"); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	res, err := appEnvRepo.GetByID(ctx, "ae-1")
	if err != nil || res == nil {
		t.Fatalf("expected application environment to exist, got err=%v res=%v", err, res)
	}
	if res.State != domain.ApplicationEnvironmentStateActive {
		t.Fatalf("expected state %q, got %q", domain.ApplicationEnvironmentStateActive, res.State)
	}
}
