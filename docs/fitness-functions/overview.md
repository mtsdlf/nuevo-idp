# Fitness Functions en el IDP

Las fitness functions son tests automatizados que verifican propiedades arquitectónicas y de negocio del IDP. Complementan los tests funcionales y las métricas.

Tipos de fitness functions que usamos:

- **Observabilidad**: garantizan que workflows y adaptadores emiten métricas y eventos con nombres estables.
- **Arquitectura**: evitan que ciertas capas dependan de otras (por ejemplo, aplicación vs adapters HTTP).
- **No funcionales** (a futuro): tiempos máximos de workflows, límites de retries, etc.

Los tests viven junto al código al que protegen, por ejemplo:

- `workflow-engine/internal/workflow/fitness_observability_test.go`
- `workflow-engine/internal/workflow/fitness_architecture_test.go`
- `control-plane-api/internal/application/fitness_architecture_test.go`

La convención es que su nombre empiece por `fitness_` y que fallen tan pronto se rompa el acuerdo que protegen.

## Cómo se ejecutan

En desarrollo y en CI se ejecutan igual que el resto de los tests de Go:

- Desde la raíz de cada módulo (por ejemplo `workflow-engine/` o `control-plane-api/`):

	```bash
	go test ./...
	```

- En CI, se recomienda tener un job por módulo que ejecute `go test ./...` y falle el pipeline si cualquier fitness function se rompe.

No requieren configuración especial: al inspeccionar código fuente y comportamiento, se ejecutan como tests normales.
