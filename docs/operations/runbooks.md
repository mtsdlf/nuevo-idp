# Runbooks de operación

Este documento recoge guías prácticas para investigar y resolver incidencias comunes en el IDP.

## SecretRotation falla o queda en estado inesperado

### Síntomas típicos

- El workflow `SecretRotation` falla repetidamente o entra en muchos retries.
- Los SecretBindings no se actualizan como se esperaba.
- El dominio indica que un secreto no está rotado aunque el proveedor (p.ej. Harbor) tenga nuevas credenciales.

### Checklist rápido

1. **Ver estado del workflow en Temporal**
   - Abrir la UI de Temporal (puerto 8233) y buscar el workflow `SecretRotation` correspondiente.
   - Revisar historial de eventos, retries y errores.

2. **Revisar métricas**
   - En Grafana (`IDP Observability` o `IDP Workflows Detail`):
     - Panel de `Secret Rotation Success Rate`.
     - Panel de duración p95 y retries para `SecretRotation`.
   - Métricas relevantes:
     - `workflow_run_duration_seconds_bucket{workflow="SecretRotation", ...}`.
     - `workflow_retries_total{workflow="SecretRotation"}`.
     - `domain_events_total{event=~"workflow_secret_rotation_.*"}`.

3. **Revisar errores hacia dependencias**
   - Panel "Provider Error Rate (downstream)" en `IDP Observability`.
   - Fijarse especialmente en entradas con `target="harbor"` o `target="execution-workers"`.

4. **Explorar trazas**
   - Desde los dashboards, usar los data links a Tempo/Jaeger.
   - Buscar trazas con `workflow.name=SecretRotation` o equivalentes.
   - Ver spans hacia `execution-workers` y, dentro de ellos, llamadas a Harbor y actualización de SecretBindings.

5. **Ver logs**
   - Inspeccionar logs de `workflow-engine` y `execution-workers`.
   - Buscar mensajes relacionados con el secreto/robot account afectado.

### Causas frecuentes

- **Errores del proveedor (Harbor)**:
  - Credenciales inválidas o permisos insuficientes.
  - Límites de rate limiting o errores 5xx.
  - Se reflejan en `downstream_errors_total{target="harbor", ...}`.

- **Problemas en actualización de SecretBindings**:
  - Handler `/secrets/bindings/update` fallando por payload inválido o estado inconsistente.
  - Errores visibles en métricas de `execution-workers` y en trazas asociadas.

- **Inconsistencias de dominio**:
  - El secreto no está en un estado rotatable según las reglas del control plane.
  - Errores de dominio devueltos por `control-plane-api` cuando el workflow intenta mutar estado.

### Acciones recomendadas

- Si el problema es de proveedor:
  - Validar credenciales y permisos fuera del IDP (p.ej. CLI de Harbor).
  - Ajustar configuración o límites según corresponda.

- Si falla la actualización de SecretBindings:
  - Revisar contrato del handler en `execution-workers`.
   - Añadir/ajustar tests de contrato si detectas cambios de payload.

- Si hay errores de dominio:
  - Revisar reglas en `control-plane-api` para el recurso afectado.
  - Corregir estado inválido vía scripts o migraciones controladas.

### Verificación de resolución

1. Re-lanzar el workflow `SecretRotation` para el secreto afectado.
2. Confirmar en Temporal que el workflow completa con éxito.
3. Verificar en Grafana que:
   - Mejora la `Secret Rotation Success Rate`.
   - No crecen más los `downstream_errors_total` asociados al caso.
4. Confirmar en el proveedor (Harbor) y en el dominio que las credenciales y SecretBindings son las esperadas.
