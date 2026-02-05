# Servicio: execution-workers

## Responsabilidad

Servicio que ejecuta side-effects e integra con sistemas externos:

- Crear repositorios (por ejemplo, en GitHub) y otros recursos de desarrollo.
- Interactuar con Harbor u otros registries para gestionar credenciales (p.ej. robot accounts).
- Actualizar SecretBindings u otros recursos derivados en función del estado de dominio.
- Normalizar errores externos a errores de dominio estables.

## Handlers y endpoints (alta vista)

Ejemplos típicos de endpoints internos (no expuestos a usuarios finales; usados por `workflow-engine` y otros servicios internos):

- `/github/repos` – creación de repositorios en GitHub.
- `/appenv/branch-protection` – aplica protección de rama (mínimo 1 aprobación de PR) sobre el repo del AppEnv.
- `/appenv/secrets` – delega en un sistema externo la materialización/configuración de secretos para el AppEnv.
- `/appenv/secret-bindings` – delega en un sistema externo la creación/actualización de bindings de secretos para el AppEnv.
- `/appenv/gitops-verify` – delega en un sistema externo la verificación de reconciliación GitOps del AppEnv.
- `/secrets/bindings/update` – desencadena la actualización de bindings de secretos tras una rotación (incluye rotación de robot token en Harbor cuando está configurado).

Todos los handlers de `/appenv/*` validan input, abren spans OTEL específicos y publican domain events (`appenv_*`) marcando `success`/`error` según el resultado del side-effect externo.

## Integraciones y proveedores

- Git provider (GitHub u otros).
- Harbor (o registry equivalente) para gestión de credenciales.

Cada integración se implementa como adapter con reglas claras de idempotencia y tratamiento de errores.

### Variables de entorno relevantes

- `GITHUB_TOKEN` – token de acceso usado por `/github/repos` y `/appenv/branch-protection`.
- `GITHUB_API_URL` – URL base opcional para GitHub Enterprise (si no se define, se usa la pública).
- `APPENV_SECRETS_ENDPOINT` – endpoint HTTP al que `/appenv/secrets` reenvía la solicitud de provisión de secretos.
- `APPENV_SECRET_BINDINGS_ENDPOINT` – endpoint HTTP al que `/appenv/secret-bindings` reenvía la creación/actualización de bindings.
- `APPENV_GITOPS_VERIFY_ENDPOINT` – endpoint HTTP al que `/appenv/gitops-verify` reenvía la verificación de reconciliación.
- `HARBOR_URL`, `HARBOR_ROBOT_USERNAME`, `HARBOR_ROBOT_PASSWORD` – configuración del cliente Harbor usado por `/secrets/bindings/update` para rotar robot tokens.
- `SECRET_BINDINGS_UPDATE_ENDPOINT` – endpoint HTTP al que `/secrets/bindings/update` reenvía la propagación de tokens rotados a todos los SecretBindings asociados a un Secret.

### Autenticación interna

- `INTERNAL_AUTH_TOKEN` – valor que debe coincidir con el configurado en `workflow-engine` y `control-plane-api`.
  - Si está definida, todos los handlers internos de `execution-workers` (`/github/repos`, `/appenv/*`, `/secrets/bindings/update`) exigen el header `X-Internal-Token` con ese valor y devuelven `401 Unauthorized` si falta o no coincide.
  - Si no está definida (modo dev/local), el servicio no aplica enforcement pero se recomienda configurarla en entornos compartidos.

## Observabilidad

- HTTP envuelto con `platform/observability.InstrumentHTTP` y trazas via OTEL.
- Métricas relevantes:
  - `http_requests_total{service="execution-workers", ...}`.
  - `downstream_errors_total{target, code, status}` para errores hacia proveedores.
- Spans con atributos que identifiquen repos, secretos, entornos, proveedor, etc., siempre sin incluir secretos.

## Tests y fitness functions

- Tests de handlers HTTP con fakes/mocks de proveedores.
- Contract tests de adapters hacia GitHub, Harbor, etc.
- Fitness functions que verifiquen:
  - Que los workers no mutan dominio directamente.
  - Que todos los errores externos se normalizan a códigos de dominio esperados.
