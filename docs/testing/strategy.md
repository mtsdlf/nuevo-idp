# Estrategia de testing

Esta sección resume cómo se testea el IDP, alineado con las prácticas definidas en `architecture.md`.

## Niveles de tests

### Dominio

- Ubicación típica: paquetes `internal/domain` o equivalentes en cada servicio.
- Características:
  - Sin IO (sin base de datos, sin HTTP, sin filesystem).
  - Sin mocks de infraestructura: se testea lógica pura.
- Objetivo: proteger reglas de negocio y estados válidos/invalidos.

### Workflows (workflow-engine)

- Enfoque: tests de integración sobre workflows usando Temporal en modo test.
- Características:
  - Se usa un test environment de Temporal.
  - Adapters externos se faken o stubbean (no se llama a GitHub real, Harbor real, etc.).
- Objetivo:
  - Validar el orden de pasos.
  - Asegurar manejo correcto de retries/timeouts.
  - Verificar que se emiten los eventos de dominio esperados.

### Adapters / Integraciones

- Enfoque: contract testing.
- Características:
  - Tests que verifican que los adapters HTTP respetan contratos (request/response) con servicios internos y externos.
  - Fakes o WireMock-like para los extremos remotos.
- Objetivo:
  - Asegurar que no se rompen los contratos cuando cambia implementación.

## Fitness functions

Además de los tests anteriores, se usan fitness functions (tests con prefijo `fitness_`) para verificar propiedades arquitectónicas y de observabilidad.

Ejemplos:

- `workflow-engine/internal/workflow/fitness_observability_test.go`:
  - Comprueba que cada workflow emite métricas y eventos de dominio con nombres esperados.
- `control-plane-api/internal/application/fitness_architecture_test.go`:
  - Verifica que la capa de dominio no depende de adapters HTTP.

Reglas generales:

- Viven junto al código que protegen.
- Se ejecutan con `go test ./...` igual que el resto de tests.
- Deben fallar tan pronto se rompa el acuerdo que resguardan.

## Cómo ejecutar tests

En local y CI se recomienda usar los objetivos del Makefile:

- Ejecutar todos los tests:

  ```bash
  make test
  ```

- Ejecutar tests por módulo:

  ```bash
  make test-api
  make test-workflow
  make test-workers
  ```

Estos comandos levantan containers de test que ejecutan `go test ./...` para cada módulo.

## Linting y calidad de código

- `make lint` ejecuta `golangci-lint run ./...` dentro de un container definido en `infra/docker-compose.yml`.
- Los linters configurados (según `architecture.md`) incluyen verificaciones de:
  - Errores no comprobados.
  - Problemas de concurrencia.
  - Reglas arquitectónicas básicas.

## Relación con observabilidad

Algunas fitness functions y tests verifican también aspectos de observabilidad, por ejemplo:

- Que ciertos eventos de dominio se emiten cuando los workflows completan.
- Que los nombres de métricas (`workflow_*`, `domain_events_total`, `downstream_errors_total`) siguen las convenciones acordadas.

Esto asegura que los dashboards y alertas se mantienen consistentes a medida que evoluciona el código.
