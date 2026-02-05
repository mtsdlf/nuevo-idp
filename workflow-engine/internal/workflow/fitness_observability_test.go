package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Estas pruebas actúan como "fitness functions" de observabilidad:
// verifican que los workflows clave emitan eventos de dominio y
// errores downstream con nombres estables. Se apoyan en inspección
// del código fuente para proteger invariantes de diseño.

func TestFitness_ApplicationOnboarding_EmitsDomainEvents(t *testing.T) {
	path := filepath.Join(".", "application_onboarding.go")
	src, err := os.ReadFile(path) //nolint:gosec // lectura de archivo local del propio módulo para fitness test
	if err != nil {
		t.Fatalf("failed to read %s: %v", path, err)
	}
	code := string(src)

	if !strings.Contains(code, "ObserveDomainEvent(\"workflow_application_onboarding_failed\"") {
		t.Errorf("expected ApplicationOnboarding to emit 'workflow_application_onboarding_failed' domain event")
	}
	if !strings.Contains(code, "ObserveDomainEvent(\"workflow_application_onboarding_completed\"") {
		t.Errorf("expected ApplicationOnboarding to emit 'workflow_application_onboarding_completed' domain event")
	}
}

func TestFitness_AppEnvProvisioning_EmitsDomainEvents(t *testing.T) {
	path := filepath.Join(".", "appenv_provisioning.go")
	src, err := os.ReadFile(path) //nolint:gosec // lectura de archivo local del propio módulo para fitness test
	if err != nil {
		t.Fatalf("failed to read %s: %v", path, err)
	}
	code := string(src)

	if !strings.Contains(code, "ObserveDomainEvent(\"workflow_appenv_provisioning_failed\"") {
		t.Errorf("expected ApplicationEnvironmentProvisioning to emit 'workflow_appenv_provisioning_failed' domain event")
	}
	if !strings.Contains(code, "ObserveDomainEvent(\"workflow_appenv_provisioning_completed\"") {
		t.Errorf("expected ApplicationEnvironmentProvisioning to emit 'workflow_appenv_provisioning_completed' domain event")
	}
}

func TestFitness_SecretRotation_EmitsDomainEvents(t *testing.T) {
	path := filepath.Join(".", "secret_rotation.go")
	src, err := os.ReadFile(path) //nolint:gosec // lectura de archivo local del propio módulo para fitness test
	if err != nil {
		t.Fatalf("failed to read %s: %v", path, err)
	}
	code := string(src)

	if !strings.Contains(code, "ObserveDomainEvent(\"workflow_secret_rotation_failed\"") {
		t.Errorf("expected SecretRotation to emit 'workflow_secret_rotation_failed' domain event")
	}
	if !strings.Contains(code, "ObserveDomainEvent(\"workflow_secret_rotation_completed\"") {
		t.Errorf("expected SecretRotation to emit 'workflow_secret_rotation_completed' domain event")
	}
}

func TestFitness_DownstreamErrors_UseObserveDownstreamError(t *testing.T) {
	path := filepath.Join(".", "appenv_provisioning.go")
	src, err := os.ReadFile(path) //nolint:gosec // lectura de archivo local del propio módulo para fitness test
	if err != nil {
		t.Fatalf("failed to read %s: %v", path, err)
	}
	code := string(src)

	if !strings.Contains(code, "ObserveDownstreamError(\"control-plane-api\"") {
		t.Errorf("expected control-plane-api downstream errors to be tracked with ObserveDownstreamError in appenv_provisioning.go")
	}
	if !strings.Contains(code, "ObserveDownstreamError(\"execution-workers\"") {
		t.Errorf("expected execution-workers downstream errors to be tracked with ObserveDownstreamError in appenv_provisioning.go")
	}
}
