package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Fitness function de arquitectura: los workflows no deben depender
// directamente de implementaciones concretas de puertos HTTP ni de //nolint:misspell // comentario en español, "implementaciones" es correcto
// otros servicios fuera de adapters inyectados desde main.

func TestFitness_Workflows_OnlyUseControlPlanePortInterface(t *testing.T) {
	path := filepath.Join(".", "application_onboarding.go")
	src, err := os.ReadFile(path) //nolint:gosec // lectura de archivo local del propio módulo para fitness test
	if err != nil {
		t.Fatalf("failed to read %s: %v", path, err)
	}
	code := string(src)

	if !strings.Contains(code, "type ApplicationOnboardingPort interface") {
		t.Errorf("expected ApplicationOnboarding to define a narrow ApplicationOnboardingPort interface")
	}

	if strings.Contains(code, "NewClient(") {
		t.Errorf("workflows must not construct concrete HTTP clients; they should receive ports via SetApplicationOnboardingPort")
	}
}
