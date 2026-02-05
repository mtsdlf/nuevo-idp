# nuevo-idp

Plataforma de desarrollo interna (IDP) basada en un control plane orientado a workflows.

## Visión rápida

- Arquitectura declarativa: ver [architecture.md](architecture.md).
- Documentación extendida: ver [docs/README.md](docs/README.md).
- Servicios principales:
  - `control-plane-api`: API HTTP y estado de dominio.
  - `workflow-engine`: orquestador Temporal de workflows.
  - `execution-workers`: ejecución de side-effects e integraciones externas.

## Arranque rápido

Requisitos:

- Docker y Docker Compose.

Comandos útiles (desde la raíz):

- Ejecutar todos los tests:

  ```bash
  make test
  ```

- Linting:

  ```bash
  make lint
  ```

- Levantar stack completo (servicios + observabilidad): ver [docs/operations/local-and-ci.md](docs/operations/local-and-ci.md).

## Dónde leer más

- Arquitectura: [docs/architecture/overview.md](docs/architecture/overview.md).
- Servicios: [docs/services](docs/services).
- Workflows: [docs/workflows/overview.md](docs/workflows/overview.md).
- Observabilidad: [docs/observability](docs/observability).
- Testing y fitness functions: [docs/testing/strategy.md](docs/testing/strategy.md).
