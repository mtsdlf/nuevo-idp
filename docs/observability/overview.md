# Observabilidad en el IDP

Este documento describe cómo está montado el stack de observabilidad y qué contratos asumimos.

## Componentes

- **Prometheus**: time-series de métricas de negocio y técnicas.
- **Tempo**: backend de trazas distribuídas (vía OTLP).
- **Grafana**: visualización de métricas y trazas; dashboard principal `IDP Observability` (uid: `idp-observability`).
- **OpenTelemetry Collector**: punto central de ingestión de spans OTLP que reenvía a Tempo.
- **Librería `platform/observability`**: helpers compartidos para métricas HTTP, workflows, eventos de dominio y errores downstream.

## Reglas generales

- Cada request HTTP y cada workflow relevante debe ser *observable* vía métricas y trazas.
- Los nombres de métricas y eventos siguen las convenciones de `architecture.md`.
- No se definen métricas "ad hoc" por servicio: todo pasa por `platform/observability`.

Para detalles de métricas concretas, ver `metrics.md`. Para trazas y Tempo, ver `tracing-tempo.md`. Para dashboards, ver `dashboards.md`.
