# Métricas y convenciones

Esta sección documenta las métricas expuestas por el IDP y cómo deben usarse.

## HTTP

Métrica principal:

- `http_requests_total{service, env, method, route, status}`
- `http_request_duration_seconds_bucket{service, env, method, route, le}` (+ `sum` y `histogram_quantile`).

Reglas:

- `service`: nombre lógico del servicio (`control-plane-api`, `workflow-engine`, `execution-workers`, ...), proviene de la env var `SERVICE_NAME`.
- `env`: entorno (`dev`, `stg`, `prod`, ...), proviene de la env var `ENVIRONMENT`.
- `route`: path normalizado (segmentos que parecen IDs se reemplazan por `{id}`) para limitar cardinalidad.
- Siempre envolver servidores HTTP con `observability.InstrumentHTTP` + `otelhttp.NewHandler`.

## Workflows (Temporal)

- `workflow_run_duration_seconds_bucket{workflow, result, le}`
- `workflow_retries_total{workflow}`

Convenciones:

- `workflow`: nombre lógico de workflow (`ApplicationOnboarding`, `ApplicationEnvironmentProvisioning`, `SecretRotation`, ...).
- `result`: `success`, `error`, `timeout`, `cancelled` o `unknown`.

## Eventos de dominio

- `domain_events_total{event, result}`

Convenciones (alineadas con `architecture.md`):

- `event`: nombre estable de caso de uso o workflow. Ejemplos:
  - `application_created`, `application_approved`.
  - `workflow_application_onboarding_completed`, `workflow_appenv_provisioning_failed`, `workflow_secret_rotation_completed`.
  - `github_repo_created`, `appenv_side_effect_accepted`, `secret_bindings_update_accepted`.
- `result`: normalmente `success` o `error`, pudiendo extenderse con `timeout`, `conflict`, etc.

## Errores hacia dependencias (downstream)

- `downstream_errors_total{target, code, status}`

Semántica:

- `target`: servicio externo lógico (`control-plane-api`, `execution-workers`, `harbor`, `git-provider`, ...).
- `code`: código de error de dominio o de cliente HTTP (p.ej. `application_invalid_state_for_onboarding`, `execution_workers_client_error`).
- `status`: status HTTP como string (`"400"`, `"404"`, `"500"`, ...).

**Regla**: todo adapter HTTP que hable con sistemas externos debe llamar a `ObserveDownstreamError` en sus caminos de error 4xx/5xx.
