package application

import (
	"context"
	"testing"

	"github.com/nuevo-idp/control-plane-api/internal/adapters/memoryrepo"
	"github.com/nuevo-idp/control-plane-api/internal/domain"
)

func TestCreateTeam_InitialStateDraft(t *testing.T) {
	teamRepo := memoryrepo.NewTeamRepository()

	services := &Services{
		Teams: teamRepo,
	}

	ctx := context.Background()
	if err := services.CreateTeam(ctx, "team-1", "Platform", "test"); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	team, err := teamRepo.GetByID(ctx, "team-1")
	if err != nil {
		t.Fatalf("unexpected error fetching team: %v", err)
	}
	if team == nil {
		t.Fatalf("expected team to be created")
	}
	if team.State != domain.TeamStateDraft {
		t.Fatalf("expected state %q, got %q", domain.TeamStateDraft, team.State)
	}
}

func TestCreateApplication_InitialStateProposedAndTeamRequired(t *testing.T) {
	teamRepo := memoryrepo.NewTeamRepository()
	appRepo := memoryrepo.NewApplicationRepository()

	services := &Services{
		Teams:        teamRepo,
		Applications: appRepo,
	}

	ctx := context.Background()
	// Need team first
	if err := services.CreateTeam(ctx, "team-1", "Platform", "test"); err != nil {
		t.Fatalf("CreateTeam failed: %v", err)
	}

	if err := services.CreateApplication(ctx, "app-1", "App", "team-1", "test"); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	app, err := appRepo.GetByID(ctx, "app-1")
	if err != nil {
		t.Fatalf("unexpected error fetching application: %v", err)
	}
	if app == nil {
		t.Fatalf("expected application to be created")
	}
	if app.State != domain.ApplicationStateProposed {
		t.Fatalf("expected state %q, got %q", domain.ApplicationStateProposed, app.State)
	}
}

func TestCreateApplication_FailsWhenTeamMissing(t *testing.T) {
	teamRepo := memoryrepo.NewTeamRepository()
	appRepo := memoryrepo.NewApplicationRepository()

	services := &Services{
		Teams:        teamRepo,
		Applications: appRepo,
	}

	ctx := context.Background()
	if err := services.CreateApplication(ctx, "app-1", "App", "missing-team", "test"); err == nil {
		t.Fatalf("expected error when team does not exist, got nil")
	}
}

func TestCreateApplication_FailsOnDuplicateID(t *testing.T) {
	teamRepo := memoryrepo.NewTeamRepository()
	appRepo := memoryrepo.NewApplicationRepository()

	services := &Services{
		Teams:        teamRepo,
		Applications: appRepo,
	}

	ctx := context.Background()
	if err := services.CreateTeam(ctx, "team-1", "Platform", "test"); err != nil {
		t.Fatalf("CreateTeam failed: %v", err)
	}

	if err := services.CreateApplication(ctx, "app-1", "App", "team-1", "test"); err != nil {
		t.Fatalf("expected first creation to succeed, got %v", err)
	}

	if err := services.CreateApplication(ctx, "app-1", "App2", "team-1", "test"); err == nil {
		t.Fatalf("expected error on duplicate application id, got nil")
	}
}
