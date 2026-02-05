# Servicio: workflow-engine

## Responsabilidad

Servicio encargado de orquestar workflows de larga duración usando Temporal:

- Coordina pasos para `ApplicationOnboarding`, `ApplicationEnvironmentProvisioning`, `SecretRotation`, etc.
- Gestiona retires, backoff y timeouts.
- Mantiene el estado de ejecución de cada workflow de forma durable.

## Workflows clave

- `ApplicationOnboarding`: onboarding de una nueva aplicación.
- `ApplicationEnvironmentProvisioning`: provisión de entornos de aplicación.
- `SecretRotation`: rotación de secretos y actualización de bindings.

Cada workflow debe estar documentado en `docs/workflows/overview.md` y alineado con los eventos de dominio.

## Integraciones

- Habla con `control-plane-api` para leer/mutar estado de dominio cuando corresponde (p.ej. marcar una aplicación como onboardeada).
- Habla con `execution-workers` para ejecutar side-effects (crear repos, rotar secretos, actualizar bindings, etc.).

Estas integraciones se realizan a través de adapters HTTP instrumentados.

### Autenticación interna

Los adapters HTTP que hablan con `control-plane-api` y `execution-workers` propagan un token interno compartido:

- Variable de entorno esperada en `workflow-engine`: `INTERNAL_AUTH_TOKEN`.
- Header que se envía en cada request interna: `X-Internal-Token`.
- Si `INTERNAL_AUTH_TOKEN` está seteada, todos los adapters la utilizan; si no, el header no se envía (modo dev/local).

## Observabilidad

- Métricas específicas de workflows:
  - `workflow_run_duration_seconds_bucket{workflow, result, ...}`.
  - `workflow_retries_total{workflow}`.
- Eventos de dominio para hitos importantes (por ejemplo, `workflow_application_onboarding_completed`, `workflow_appenv_provisioning_failed`).
- Trazas con un span por paso de workflow y spans anidados para llamadas a otros servicios.

## Tests y fitness functions

- Tests de workflows con `Temporal` en modo test (test env) y fakes para adapters.
- Tests de contrato sobre adapters HTTP hacia `control-plane-api` y `execution-workers`.
- Fitness functions que aseguran:
  - Que los nombres de workflows y eventos de dominio siguen las convenciones de `architecture.md`.
  - Que los workflows no contienen lógica de acceso directo a proveedores externos.
