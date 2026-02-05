# workflow-engine

Servicio responsable de orquestar workflows de larga duración usando Temporal.

Workflows principales:

- `ApplicationOnboarding`
- `ApplicationEnvironmentProvisioning`
- `SecretRotation`

## Ejecutar en local

Requisitos:

- Temporal corriendo (puede levantarse con `infra/docker-compose.yml`).

Ejemplo rápido:

```bash
cd workflow-engine
go test ./...
# go run ./cmd/engine  # requiere Temporal y dependencias
```

Para levantar el stack completo (incluyendo Temporal, API y workers), ver [docs/operations/local-and-ci.md](../docs/operations/local-and-ci.md).

## Tests

```bash
cd workflow-engine
go test ./...
```

En CI/local vía Docker Compose:

```bash
make test-workflow
```

## Más detalles

- Descripción de workflows y responsabilidades: [docs/services/workflow-engine.md](../docs/services/workflow-engine.md).
- Workflows clave: [docs/workflows/overview.md](../docs/workflows/overview.md).
