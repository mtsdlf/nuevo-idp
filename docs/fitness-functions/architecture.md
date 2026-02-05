# Fitness Functions de Arquitectura

## Objetivo

Proteger las fronteras entre capas (dominio, aplicación, workflows, adapters) y entre servicios (`control-plane-api`, `workflow-engine`, `execution-workers`).

## Tests actuales

1. `control-plane-api/internal/application/fitness_architecture_test.go`

   - `TestFitness_ApplicationServices_DoNotDependOnHTTPAdapters`:
     - Inspecciona `services.go` y falla si aparecen imports a:
       - `internal/adapters/httpapi`.
       - `net/http`.
     - Garantiza que la capa de aplicación no se acopla a detalles de transporte ni a adapters HTTP concretos.

2. `workflow-engine/internal/workflow/fitness_architecture_test.go`

   - `TestFitness_Workflows_OnlyUseControlPlanePortInterface`:
     - Verifica que `application_onboarding.go` define la interfaz `ApplicationOnboardingPort`.
     - Falla si el código construye directamente clientes HTTP (`NewClient(...)`).
     - Protege el patrón puertos/adapters: los workflows orquestan usando puertos, no implementaciones concretas.

## Cómo extender

- Para nuevos servicios o capas, identifica qué dependencias están prohibidas y añade tests que:
  - Lean el archivo fuente.
  - Verifiquen que no aparecen imports prohibidos.

- Ejemplos futuros:
  - Asegurar que `execution-workers` no importa paquetes de dominio de `control-plane-api`.
  - Asegurar que `internal/domain` no importa `platform/observability` ni `net/http`.

Estas fitness functions deben ejecutarse en CI como parte de `go test ./...` para frenar cambios que violen la arquitectura declarada en `architecture.md`.
