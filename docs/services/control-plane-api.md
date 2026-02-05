# Servicio: control-plane-api

## Responsabilidad

Servicio frontal que expone la API HTTP del IDP:

- Recibe comandos de negocio (crear aplicación, provisionar entorno, solicitar rotación de secretos, etc.).
- Valida reglas de dominio.
- Mutar el estado de dominio persistido.
- Emite eventos de dominio que describen lo que ocurrió.
- Sirve consultas de lectura sobre el estado de las aplicaciones y sus entornos.

No ejecuta side-effects ni llama proveedores externos: delega esas acciones a workflows y workers.

## Endpoints (alta vista)

- Comandos principales (ejemplos):
  - `POST /applications` – crear/registrar aplicación.
  - `POST /applications/{id}/environments` – provisionar entorno.
  - `POST /secrets/{id}/rotation` – solicitar rotación de secreto.

- Consultas principales: dependen del modelo actual, pero típicamente incluyen:
  - `GET /applications` / `GET /applications/{id}`.
  - `GET /applications/{id}/environments`.

Los detalles exactos de payloads y errores deben mantenerse sincronizados con los handlers HTTP dentro del módulo `control-plane-api`.

## Observabilidad

- HTTP envuelto con `platform/observability.InstrumentHTTP` + `otelhttp.NewHandler`.
- Métricas clave:
  - `http_requests_total{service="control-plane-api", ...}`.
  - `http_request_duration_seconds_bucket{service="control-plane-api", ...}`.
  - Eventos de dominio emitidos vía `domain_events_total`.

- Trazas:
  - Un trace por request.
  - Spans para validaciones importantes o llamadas internas hacia el `workflow-engine`.

## Tests y fitness functions

- Tests de dominio: lógica de negocio sin IO.
- Tests HTTP: validan contratos de la API (status codes, shape de errores, etc.).
- Fitness functions (ejemplos):
  - Proteger que la capa de dominio no depende de adapters HTTP.
  - Asegurar que los errores de dominio se mapean a HTTP según lo acordado.

## Relaciones con otros componentes

- Llama al `workflow-engine` (vía HTTP/SDK según implementación) para iniciar workflows (`ApplicationOnboarding`, `ApplicationEnvironmentProvisioning`, `SecretRotation`).
- No habla directamente con `execution-workers` ni con proveedores externos.

## Autenticación interna

Las llamadas que vienen del `workflow-engine` hacia los comandos internos de `control-plane-api` se protegen con un token compartido:

- Variable de entorno: `INTERNAL_AUTH_TOKEN`.
- Header esperado en requests internas: `X-Internal-Token`.
- Si `INTERNAL_AUTH_TOKEN` está configurada, los handlers internos devuelven `401 Unauthorized` cuando el header falta o no coincide.
- Si no está configurada (modo dev/local), el servicio acepta la request sin enforcement, pero se recomienda definirla en entornos compartidos.
