# control-plane-api

Servicio HTTP frontal del IDP. Se encarga de:

- Recibir comandos de negocio (crear aplicación, provisionar entorno, rotar secretos, etc.).
- Validar reglas de dominio.
- Mutar el estado de dominio persistido.
- Emitir eventos de dominio.
- Servir consultas sobre aplicaciones y entornos.

## Ejecutar en local

Requisitos:

- Go 1.21+ (para ejecución directa), o Docker si usas `infra/docker-compose.yml`.

Ejemplo rápido (sólo servicio, sin dependencias):

```bash
cd control-plane-api
go test ./...
# go run ./cmd/api  # requiere DB configurada
```

Para un entorno completo (Postgres, Temporal, otros servicios), ver [docs/operations/local-and-ci.md](../docs/operations/local-and-ci.md).

## Tests

```bash
cd control-plane-api
go test ./...
```

En CI/local vía Docker Compose:

```bash
make test-api
```

## Más detalles

- Responsabilidades y límites: [docs/services/control-plane-api.md](../docs/services/control-plane-api.md).
- Arquitectura general: [docs/architecture/overview.md](../docs/architecture/overview.md).
