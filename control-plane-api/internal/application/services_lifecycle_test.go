package application

import (
	"context"
	"testing"

	"github.com/nuevo-idp/control-plane-api/internal/adapters/memoryrepo"
	"github.com/nuevo-idp/control-plane-api/internal/domain"
)

func TestApproveApplication_TransitionsProposedToApproved(t *testing.T) {
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
		t.Fatalf("CreateApplication failed: %v", err)
	}

	if err := services.ApproveApplication(ctx, "app-1", "approver"); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	app, err := appRepo.GetByID(ctx, "app-1")
	if err != nil || app == nil {
		t.Fatalf("expected application, got err=%v app=%v", err, app)
	}
	if app.State != domain.ApplicationStateApproved {
		t.Fatalf("expected state %q, got %q", domain.ApplicationStateApproved, app.State)
	}
}

func TestApproveApplication_FailsFromNonProposed(t *testing.T) {
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
		t.Fatalf("CreateApplication failed: %v", err)
	}

	// Forzamos a Active para simular que ya pasó por otros workflows
	app, err := appRepo.GetByID(ctx, "app-1")
	if err != nil || app == nil {
		t.Fatalf("expected app, got err=%v app=%v", err, app)
	}
	app.State = domain.ApplicationStateActive
	if err := appRepo.Save(ctx, app); err != nil {
		t.Fatalf("saving app failed: %v", err)
	}

	if err := services.ApproveApplication(ctx, "app-1", "approver"); err == nil {
		t.Fatalf("expected error when approving from non-Proposed state, got nil")
	}
}

func TestDeprecateApplication_TransitionsActiveToDeprecated(t *testing.T) {
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
		t.Fatalf("CreateApplication failed: %v", err)
	}

	// Simulamos que la app ya está Active
	app, err := appRepo.GetByID(ctx, "app-1")
	if err != nil || app == nil {
		t.Fatalf("expected app, got err=%v app=%v", err, app)
	}
	app.State = domain.ApplicationStateActive
	if err := appRepo.Save(ctx, app); err != nil {
		t.Fatalf("saving app failed: %v", err)
	}

	if err := services.DeprecateApplication(ctx, "app-1", "user"); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	updated, err := appRepo.GetByID(ctx, "app-1")
	if err != nil || updated == nil {
		t.Fatalf("expected app, got err=%v app=%v", err, updated)
	}
	if updated.State != domain.ApplicationStateDeprecated {
		t.Fatalf("expected state %q, got %q", domain.ApplicationStateDeprecated, updated.State)
	}
}

func TestDeprecateApplication_FailsFromNonActive(t *testing.T) {
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
		t.Fatalf("CreateApplication failed: %v", err)
	}

	if err := services.DeprecateApplication(ctx, "app-1", "user"); err == nil {
		t.Fatalf("expected error when deprecating from non-Active state, got nil")
	}
}

func TestStartApplicationOnboarding_TransitionsApprovedToOnboarding(t *testing.T) {
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
		t.Fatalf("CreateApplication failed: %v", err)
	}

	// Precondición: Approved
	if err := services.ApproveApplication(ctx, "app-1", "approver"); err != nil {
		t.Fatalf("ApproveApplication failed: %v", err)
	}

	if err := services.StartApplicationOnboarding(ctx, "app-1", "user"); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	updated, err := appRepo.GetByID(ctx, "app-1")
	if err != nil || updated == nil {
		t.Fatalf("expected app, got err=%v app=%v", err, updated)
	}
	if updated.State != domain.ApplicationStateOnboarding {
		t.Fatalf("expected state %q, got %q", domain.ApplicationStateOnboarding, updated.State)
	}
}

func TestStartApplicationOnboarding_FailsFromNonApproved(t *testing.T) {
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
		t.Fatalf("CreateApplication failed: %v", err)
	}

	// Sigue en Proposed
	if err := services.StartApplicationOnboarding(ctx, "app-1", "user"); err == nil {
		t.Fatalf("expected error when starting onboarding from non-Approved state, got nil")
	}
}

func TestActivateApplication_TransitionsOnboardingToActive(t *testing.T) {
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
		t.Fatalf("CreateApplication failed: %v", err)
	}
	if err := services.ApproveApplication(ctx, "app-1", "approver"); err != nil {
		t.Fatalf("ApproveApplication failed: %v", err)
	}
	if err := services.StartApplicationOnboarding(ctx, "app-1", "user"); err != nil {
		t.Fatalf("StartApplicationOnboarding failed: %v", err)
	}

	if err := services.ActivateApplication(ctx, "app-1", "activator"); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	updated, err := appRepo.GetByID(ctx, "app-1")
	if err != nil || updated == nil {
		t.Fatalf("expected app, got err=%v app=%v", err, updated)
	}
	if updated.State != domain.ApplicationStateActive {
		t.Fatalf("expected state %q, got %q", domain.ApplicationStateActive, updated.State)
	}
}

func TestActivateApplication_FailsFromNonOnboarding(t *testing.T) {
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
		t.Fatalf("CreateApplication failed: %v", err)
	}

	// Sigue en Proposed
	if err := services.ActivateApplication(ctx, "app-1", "activator"); err == nil {
		t.Fatalf("expected error when activating from non-Onboarding state, got nil")
	}
}

func TestStartSecretRotation_TransitionsActiveToRotating(t *testing.T) { //nolint:dupl,misspell // setup deliberadamente similar a CompleteSecretRotation para cubrir ambas transiciones de estado
	teamRepo := memoryrepo.NewTeamRepository()
	secretRepo := memoryrepo.NewSecretRepository()

	services := &Services{
		Teams:   teamRepo,
		Secrets: secretRepo,
	}

	ctx := context.Background()
	if err := services.CreateTeam(ctx, "team-1", "Platform", "test"); err != nil {
		t.Fatalf("CreateTeam failed: %v", err)
	}
	if err := services.CreateSecret(ctx, "sec-1", "team-1", "runtime", "high", "test"); err != nil {
		t.Fatalf("CreateSecret failed: %v", err)
	}

	// Simulamos que el secreto está Active
	sec, err := secretRepo.GetByID(ctx, "sec-1")
	if err != nil || sec == nil {
		t.Fatalf("expected secret, got err=%v sec=%v", err, sec)
	}
	sec.State = domain.SecretStateActive
	if err := secretRepo.Save(ctx, sec); err != nil {
		t.Fatalf("saving secret failed: %v", err)
	}

	if err := services.StartSecretRotation(ctx, "sec-1", "user"); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	updated, err := secretRepo.GetByID(ctx, "sec-1")
	if err != nil || updated == nil {
		t.Fatalf("expected secret, got err=%v sec=%v", err, updated)
	}
	if updated.State != domain.SecretStateRotating {
		t.Fatalf("expected state %q, got %q", domain.SecretStateRotating, updated.State)
	}
}

func TestStartSecretRotation_FailsFromNonActive(t *testing.T) {
	teamRepo := memoryrepo.NewTeamRepository()
	secretRepo := memoryrepo.NewSecretRepository()

	services := &Services{
		Teams:   teamRepo,
		Secrets: secretRepo,
	}

	ctx := context.Background()
	if err := services.CreateTeam(ctx, "team-1", "Platform", "test"); err != nil {
		t.Fatalf("CreateTeam failed: %v", err)
	}
	if err := services.CreateSecret(ctx, "sec-1", "team-1", "runtime", "high", "test"); err != nil {
		t.Fatalf("CreateSecret failed: %v", err)
	}

	// Sigue en Declared
	if err := services.StartSecretRotation(ctx, "sec-1", "user"); err == nil {
		t.Fatalf("expected error when starting rotation from non-Active state, got nil")
	}
}

func TestCompleteSecretRotation_TransitionsRotatingToActive(t *testing.T) { //nolint:dupl,misspell // comparte setup con StartSecretRotation pero valida transición opuesta
	teamRepo := memoryrepo.NewTeamRepository()
	secretRepo := memoryrepo.NewSecretRepository()

	services := &Services{
		Teams:   teamRepo,
		Secrets: secretRepo,
	}

	ctx := context.Background()
	if err := services.CreateTeam(ctx, "team-1", "Platform", "test"); err != nil {
		t.Fatalf("CreateTeam failed: %v", err)
	}
	if err := services.CreateSecret(ctx, "sec-1", "team-1", "runtime", "high", "test"); err != nil {
		t.Fatalf("CreateSecret failed: %v", err)
	}

	// Pasamos el secreto a Rotating
	sec, err := secretRepo.GetByID(ctx, "sec-1")
	if err != nil || sec == nil {
		t.Fatalf("expected secret, got err=%v sec=%v", err, sec)
	}
	sec.State = domain.SecretStateRotating
	if err := secretRepo.Save(ctx, sec); err != nil {
		t.Fatalf("saving secret failed: %v", err)
	}

	if err := services.CompleteSecretRotation(ctx, "sec-1", "user"); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	updated, err := secretRepo.GetByID(ctx, "sec-1")
	if err != nil || updated == nil {
		t.Fatalf("expected secret, got err=%v sec=%v", err, updated)
	}
	if updated.State != domain.SecretStateActive {
		t.Fatalf("expected state %q, got %q", domain.SecretStateActive, updated.State)
	}
}

func TestCompleteSecretRotation_FailsFromNonRotating(t *testing.T) {
	teamRepo := memoryrepo.NewTeamRepository()
	secretRepo := memoryrepo.NewSecretRepository()

	services := &Services{
		Teams:   teamRepo,
		Secrets: secretRepo,
	}

	ctx := context.Background()
	if err := services.CreateTeam(ctx, "team-1", "Platform", "test"); err != nil {
		t.Fatalf("CreateTeam failed: %v", err)
	}
	if err := services.CreateSecret(ctx, "sec-1", "team-1", "runtime", "high", "test"); err != nil {
		t.Fatalf("CreateSecret failed: %v", err)
	}

	// Sigue en Declared
	if err := services.CompleteSecretRotation(ctx, "sec-1", "user"); err == nil {
		t.Fatalf("expected error when completing rotation from non-Rotating state, got nil")
	}
}
