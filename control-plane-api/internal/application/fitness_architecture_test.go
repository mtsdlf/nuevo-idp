package application

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Fitness function de arquitectura: los servicios de aplicación sólo
// pueden depender del paquete de dominio y de la plataforma compartida,
// nunca de adapters HTTP específicos u otros servicios.

func TestFitness_ApplicationServices_DoNotDependOnHTTPAdapters(t *testing.T) {
	path := filepath.Join(".", "services.go")
	src, err := os.ReadFile(path) //nolint:gosec // lectura de archivo local del propio módulo para fitness test
	if err != nil {
		t.Fatalf("failed to read %s: %v", path, err)
	}
	code := string(src)

	forbiddenImports := []string{
		"internal/adapters/httpapi",
		"net/http",
	}

	for _, imp := range forbiddenImports {
		if strings.Contains(code, "\""+imp+"\"") {
			t.Errorf("application services must not import %q (evitar acoplar dominio a adapters)", imp)
		}
	}
}
