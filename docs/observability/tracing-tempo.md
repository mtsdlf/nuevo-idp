# Trazas y Tempo

## Flujo de spans

1. Los servicios usan OpenTelemetry SDK (configurado vía `platform/tracing`).
2. Cada proceso llama a `InitTracing(ctx, serviceName)` en `main`.
3. Las apps exportan hacia `OTEL_EXPORTER_OTLP_ENDPOINT=http://otel-collector:4318` (HTTP).
4. El OpenTelemetry Collector reenvía las trazas a Tempo vía OTLP gRPC (`tempo:4317`).
5. Grafana consulta Tempo mediante el datasource `Tempo` (`http://tempo:3200`).

## Convenciones de spans

- Un trace por request HTTP entrante o por ejecución de workflow Temporal.
- Un span por step de workflow y por llamada a sistemas externos.
- Atributos recomendados por dominio:
  - `idp.application_id`, `idp.application_environment_id`, `idp.secret_id`, `idp.team_id`.
  - `http.route`, `http.method`, `http.status_code` para spans HTTP.

## Navegación desde Grafana

El dashboard `IDP Observability` incluye data links "Ver trazas en Tempo" que abren Grafana Explore con:

- Datasource: `Tempo`.
- Query basada en `service.name="$service"` (label estándar OTEL).

Esto permite saltar rápidamente de picos de latencia/error a las trazas asociadas.
