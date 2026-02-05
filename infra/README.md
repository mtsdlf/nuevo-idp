# Infraestructura

Este directorio contiene la infraestructura de apoyo para desarrollo y observabilidad del IDP.

## Componentes principales

- `docker-compose.yml`:
  - Postgres.
  - Temporal (servidor + UI).
  - Servicios de la app: `control-plane-api`, `workflow-engine`, `execution-workers`.
  - Observabilidad: OTEL Collector, Tempo, Jaeger, Prometheus, Grafana.
- `init-db.sql`: inicialización básica de la base de datos.
- `prometheus/`: configuración de Prometheus.
- `grafana/`: dashboards y datasources provisionados.
- `tempo/`: configuración del backend de trazas.

## Uso

Ver guía detallada en [docs/operations/local-and-ci.md](../docs/operations/local-and-ci.md).

Ejemplo rápido para levantar el stack completo:

```bash
cd infra
docker compose -f docker-compose.yml up --build
```
