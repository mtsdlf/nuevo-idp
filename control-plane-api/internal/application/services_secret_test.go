package application

import (
	"context"
	"testing"

	"github.com/nuevo-idp/control-plane-api/internal/adapters/memoryrepo"
	"github.com/nuevo-idp/control-plane-api/internal/domain"
)

func TestCreateSecret_RequiresOwnerTeamAndStartsDeclared(t *testing.T) {
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
		t.Fatalf("expected no error, got %v", err)
	}

	sec, err := secretRepo.GetByID(ctx, "sec-1")
	if err != nil || sec == nil {
		t.Fatalf("expected secret to be created, got err=%v sec=%v", err, sec)
	}
	if sec.State != domain.SecretStateDeclared {
		t.Fatalf("expected state %q, got %q", domain.SecretStateDeclared, sec.State)
	}
	if sec.OwnerTeam != "team-1" {
		t.Fatalf("expected owner team 'team-1', got %q", sec.OwnerTeam)
	}
}

func TestCreateSecret_FailsWithoutOwnerTeam(t *testing.T) {
	teamRepo := memoryrepo.NewTeamRepository()
	secretRepo := memoryrepo.NewSecretRepository()

	services := &Services{
		Teams:   teamRepo,
		Secrets: secretRepo,
	}

	ctx := context.Background()
	if err := services.CreateSecret(ctx, "sec-1", "", "runtime", "high", "test"); err == nil {
		t.Fatalf("expected error when owner team is empty, got nil")
	}
}

func TestDeclareSecretBinding_RequiresActiveSecret(t *testing.T) {
	teamRepo := memoryrepo.NewTeamRepository()
	secretRepo := memoryrepo.NewSecretRepository()
	bindingRepo := memoryrepo.NewSecretBindingRepository()

	services := &Services{
		Teams:          teamRepo,
		Secrets:        secretRepo,
		SecretBindings: bindingRepo,
	}

	ctx := context.Background()
	if err := services.CreateTeam(ctx, "team-1", "Platform", "test"); err != nil {
		t.Fatalf("CreateTeam failed: %v", err)
	}

	// Creamos un secreto declarado y lo dejamos en Declared (no Active)
	if err := services.CreateSecret(ctx, "sec-1", "team-1", "runtime", "high", "test"); err != nil {
		t.Fatalf("CreateSecret failed: %v", err)
	}

	if err := services.DeclareSecretBinding(ctx, "bind-1", "sec-1", "target-1", "CodeRepository", "test"); err == nil {
		t.Fatalf("expected error because secret is not Active, got nil")
	}

	// Forzamos el secreto a Active para simular que el workflow de rotación lo activó
	sec, err := secretRepo.GetByID(ctx, "sec-1")
	if err != nil || sec == nil {
		t.Fatalf("expected secret to exist, got err=%v sec=%v", err, sec)
	}
	sec.State = domain.SecretStateActive
	if err := secretRepo.Save(ctx, sec); err != nil {
		t.Fatalf("saving updated secret failed: %v", err)
	}

	if err := services.DeclareSecretBinding(ctx, "bind-1", "sec-1", "target-1", "CodeRepository", "test"); err != nil {
		t.Fatalf("expected binding to succeed with active secret, got %v", err)
	}
}
