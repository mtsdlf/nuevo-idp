package internal

import (
    "os"
    "path/filepath"
    "strings"
    "testing"
)

// Fitness function de arquitectura: execution-workers no debe depender de
// otros servicios del monorepo (control-plane-api, workflow-engine).
func TestFitness_Workers_DoNotDependOnOtherServices(t *testing.T) {
    // Usamos la raíz del módulo (un nivel arriba de internal) para localizar
    // el main de los workers de forma robusta cuando go test ejecuta en
    // el directorio internal.
    path := filepath.Join("..", "cmd", "worker", "main.go")
    src, err := os.ReadFile(path) //nolint:gosec // lectura de archivo local dentro del propio módulo para fitness test
    if err != nil {
        t.Fatalf("failed to read %s: %v", path, err)
    }
    code := string(src)

    forbidden := []string{
        "\"github.com/nuevo-idp/control-plane-api\"",
        "\"github.com/nuevo-idp/workflow-engine\"",
    }

    for _, imp := range forbidden {
        if strings.Contains(code, imp) {
            t.Errorf("execution-workers must not import %s (evitar acoplar workers a otros servicios)", imp)
        }
    }
}
