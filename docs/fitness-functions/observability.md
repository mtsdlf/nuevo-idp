# Fitness Functions de Observabilidad

## Objetivo

Asegurar que los workflows y adaptadores clave mantienen sus contratos de observabilidad:

- Emitir `domain_events_total` con nombres estables.
- Usar `ObserveDownstreamError` para errores hacia sistemas externos.

## Tests actuales

Archivo: `workflow-engine/internal/workflow/fitness_observability_test.go`.

Verifica que:

1. **ApplicationOnboarding**
   - Emite `workflow_application_onboarding_failed`.
   - Emite `workflow_application_onboarding_completed`.

2. **ApplicationEnvironmentProvisioning**
   - Emite `workflow_appenv_provisioning_failed`.
   - Emite `workflow_appenv_provisioning_completed`.

3. **SecretRotation**
   - Emite `workflow_secret_rotation_failed`.
   - Emite `workflow_secret_rotation_completed`.

4. **Errores downstream**
   - `appenv_provisioning.go` usa `ObserveDownstreamError("control-plane-api", ...)`.
   - `appenv_provisioning.go` usa `ObserveDownstreamError("execution-workers", ...)`.

Estos tests inspeccionan el código fuente (no sólo el comportamiento) para garantizar que los nombres de eventos no se pierden ni se renombrar sin intención.

## Cómo extender

- Cuando agregues un nuevo workflow relevante, define sus eventos `workflow_<aggregate>_<flow>_(completed|failed)` y añade asserts equivalentes en este archivo.
- Si añades un nuevo adapter HTTP hacia un sistema externo, añade asserts que verifiquen el uso de `ObserveDownstreamError` con el `target` correcto.
