# execution-workers

Servicio encargado de ejecutar side-effects e integrarse con sistemas externos (GitHub, Harbor, etc.).

Ejemplos de responsabilidades:

- Crear repositorios.
- Gestionar credenciales en Harbor u otros registries.
- Actualizar SecretBindings u otros recursos derivados.

## Ejecutar en local

Requisitos:

- Temporal corriendo.
- Variables de entorno para proveedores (por ejemplo `GITHUB_TOKEN`).

Ejemplo rápido:

```bash
cd execution-workers
go test ./...
# go run ./cmd/worker  # requiere Temporal y configuración de proveedores
```

En un entorno completo, se recomienda usar `infra/docker-compose.yml`. Ver [docs/operations/local-and-ci.md](../docs/operations/local-and-ci.md).

## Tests

```bash
cd execution-workers
go test ./...
```

En CI/local vía Docker Compose:

```bash
make test-workers
```

## Más detalles

- Responsabilidades, endpoints internos e integraciones: [docs/services/execution-workers.md](../docs/services/execution-workers.md).
