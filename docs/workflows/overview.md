# Workflows del IDP

Este documento resume los workflows principales del IDP y cómo se relacionan con métricas, eventos de dominio y trazas.

Workflows principales (según `architecture.md`):

- `ApplicationOnboarding`
- `ApplicationEnvironmentProvisioning`
- `SecretRotation`

## ApplicationOnboarding

### Intención

Onboardear una nueva aplicación al IDP, dejando lista la base para futuros entornos, repos y credenciales.

### Trigger

- Comando HTTP al `control-plane-api` para crear/registrar una aplicación.

### Pasos típicos (alto nivel)

1. Validar reglas de dominio (que la aplicación puede ser onboardeada).
2. Mutar estado de dominio en el `control-plane-api` (persistir aplicación, marcar estado inicial).
3. Emitir eventos de dominio tipo `application_created`, `workflow_application_onboarding_completed` / `failed`.
4. Orquestar side-effects necesarios vía `workflow-engine` + `execution-workers` (p.ej. creación de repos, semillas de config inicial, etc.).

### Métricas y eventos

- `workflow_run_duration_seconds_bucket{workflow="ApplicationOnboarding", result=...}`
- `workflow_retries_total{workflow="ApplicationOnboarding"}`
- `domain_events_total{event=~"workflow_application_onboarding_.*", result=...}`

Esto permite responder:

- ¿Qué tan seguido se ejecuta el onboarding?
- ¿Cuál es su latencia p95?
- ¿Qué tasa de éxito/fracaso tiene?

## ApplicationEnvironmentProvisioning

### Intención

Provisionar un entorno de aplicación (por ejemplo, `dev` o `prod`) incluyendo infra básica, bindings y configuraciones necesarias.

### Trigger

- Comando HTTP al `control-plane-api` para provisionar un nuevo entorno de aplicación.

### Pasos típicos (alto nivel)

1. Validar que la aplicación y el entorno están en un estado válido para provisionar.
2. Actualizar el estado de dominio (entorno en estado "provisioning" / similar).
3. Orquestar side-effects vía `workflow-engine`:
	- `MaterializeRepositories` → usa un `GitProvider` (adapter HTTP a `execution-workers`) para crear el repo `appenv-<ApplicationEnvironmentID>`.
	- `ApplyBranchProtection` → llama al `AppEnvProvisioningProvider` que a su vez invoca `/appenv/branch-protection` en `execution-workers`.
	- `ProvisionSecrets` → llama al `AppEnvProvisioningProvider` → `/appenv/secrets` en `execution-workers`.
	- `CreateSecretBindings` → llama al `AppEnvProvisioningProvider` → `/appenv/secret-bindings` en `execution-workers`.
	- `VerifyGitOpsReconciliation` → llama al `AppEnvProvisioningProvider` → `/appenv/gitops-verify` en `execution-workers`.
	- `FinalizeApplicationEnvironmentProvisioning` → notifica al `control-plane-api` que el entorno quedó provisionado.
4. Emitir eventos `workflow_appenv_provisioning_completed` / `failed` según resultado.

### Métricas y eventos

- `workflow_run_duration_seconds_bucket{workflow="ApplicationEnvironmentProvisioning", result=...}`
- `workflow_retries_total{workflow="ApplicationEnvironmentProvisioning"}`
- `domain_events_total{event=~"workflow_appenv_provisioning_.*", result=...}`

## SecretRotation

### Intención

Rotar secretos de forma segura, coordinando credenciales en proveedores (p.ej. Harbor) y bindings en aplicaciones/entornos.

### Trigger

- Comando HTTP al `control-plane-api` o evento interno que dispare la rotación para un secreto concreto.

### Pasos típicos (alto nivel)

1. Validar que el secreto es rotatable y no está en uso por otro flujo crítico.
2. Solicitar a `execution-workers` la rotación en el proveedor (por ejemplo, token de robot en Harbor).
3. Actualizar los SecretBindings vía `execution-workers`.
4. Actualizar el estado de dominio marcando el secreto como rotado.
5. Emitir eventos `secret_rotation_started`, `workflow_secret_rotation_completed` / `failed`.

### Métricas y eventos

- `workflow_run_duration_seconds_bucket{workflow="SecretRotation", result=...}`
- `workflow_retries_total{workflow="SecretRotation"}`
- `domain_events_total{event=~"workflow_secret_rotation_.*"}`

## Relación con observabilidad

Cada workflow relevante debe:

- Generar un *trace* con un span por paso importante (según `observability.tracing` en `architecture.md`).
- Emitir eventos de dominio (`domain_events_total`) siguiendo la convención `workflow_<aggregate>_<flow>`.
- Emitir métricas de duración y retries de forma uniforme.

Los dashboards `IDP Observability` y `IDP Workflows Detail` consumen estas métricas para dar visibilidad operativa end-to-end.
