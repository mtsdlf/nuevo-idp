# Operación local y en CI

Este documento describe cómo levantar el entorno del IDP en local y cómo se ejecutan tests/lint desde CI usando Docker Compose y el Makefile de la raíz.

## Stack local (docker compose)

El archivo principal es `infra/docker-compose.yml`. Los servicios clave son:

- **postgres**: base de datos `idp` para el control plane.
- **temporal**: servidor Temporal + UI (puerto 8233) usando Postgres como backend.
- **control-plane-api**: API HTTP del IDP (puerto 8080).
- **workflow-engine**: motor de workflows (puerto 8081, health/metrics HTTP).
- **execution-workers**: workers para side-effects.
- **observabilidad**:
  - `otel-collector`: recibe spans OTLP.
  - `tempo`: backend de trazas, consumido por Grafana.
  - `jaeger`: UI opcional de trazas.
  - `prometheus`: scraping de métricas.
  - `grafana`: dashboards en puerto 3000.

### Levantar sólo la infraestructura base

Desde la raíz del repositorio:

```bash
cd infra
docker compose -f docker-compose.yml up postgres temporal
```

Puedes añadir `prometheus`, `grafana`, `tempo` o `jaeger` según lo que necesites inspeccionar.

### Levantar toda la solución (servicios + observabilidad)

```bash
cd infra
docker compose -f docker-compose.yml up --build postgres temporal control-plane-api workflow-engine execution-workers otel-collector tempo jaeger prometheus grafana
```

Requisitos:

- Docker y Docker Compose instalados.
- Variable `GITHUB_TOKEN` disponible en el entorno si quieres probar flujos que tocan GitHub.

## Tests y lint vía Docker Compose

Los servicios `*-tests` y `golangci-lint` del `docker-compose.yml` ejecutan `go test ./...` y linting dentro de containers con el código montado.

El Makefile de la raíz expone atajos:

- `make test` – ejecuta tests de los tres módulos:
  - `control-plane-api-tests`
  - `workflow-engine-tests`
  - `execution-workers-tests`
- `make test-api` – sólo `control-plane-api`.
- `make test-workflow` – sólo `workflow-engine`.
- `make test-workers` – sólo `execution-workers`.
- `make lint` – corre `golangci-lint run ./...` en el código.

Internamente estos comandos ejecutan:

```bash
cd infra && docker compose -f docker-compose.yml up --build --abort-on-container-exit <service>
```

## Integración en CI

En CI se recomienda reutilizar exactamente estos comandos, por ejemplo:

- Job de tests:
  - `make test`
- Job de lint:
  - `make lint`

Beneficios:

- Asegura paridad entre entorno local y CI (misma imagen de Go, mismas dependencias).
- Minimiza diferencias de configuración entre desarrolladores.

## Puertos relevantes

- Postgres: `5432`.
- Temporal gRPC: `7233`.
- Temporal UI: `8233`.
- control-plane-api: `8080`.
- workflow-engine: `8081`.
- Prometheus: `9090`.
- Grafana: `3000`.
- Jaeger UI: `16686`.
- Tempo HTTP API: `3200`.
