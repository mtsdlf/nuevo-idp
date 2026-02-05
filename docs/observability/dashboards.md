# Dashboards en Grafana

## Dashboard principal: IDP Observability

UID: `idp-observability`
Archivo: `infra/grafana/dashboards/observability.json`.

### Secciones

1. **Visión general HTTP**
   - Error Rate (5xx) por servicio.
   - Throughput (req/s) global.
   - Latencia p95 global.
   - Filtros por `service` y `env` (variables de dashboard).

2. **HTTP por ruta**
   - Requests por `service, route, method`.
   - Latencia p95 por `service, route, method`.

3. **KPIs de negocio (workflows)**
   - Application Onboarding Success Rate.
   - AppEnv Provisioning Success Rate.
   - Secret Rotation Success Rate.

4. **Workflows detallados (visión global)**
   - Duración p95 por workflow (`workflow_run_duration_seconds_bucket`).
   - Retries por workflow (`workflow_retries_total`).

5. **Dependencias y eventos de dominio**
   - Provider Error Rate (`downstream_errors_total`).
   - Domain Events (`domain_events_total`).

### Data links

Algunos panels incluyen data links a:

- **Tempo** (trazas): `"Ver trazas en Tempo"` abre Explore con `service.name="$service"`.
- **Prometheus** (Explore): queries preconstruidas sobre las mismas métricas del panel.

## Dashboard de detalle de workflows: IDP Workflows Detail

UID: `idp-workflows-detail`
Archivo: `infra/grafana/dashboards/workflows-detail.json`.

### Secciones

1. **Application Onboarding**
   - Success Rate del workflow.
   - Duración p95 del workflow.
   - Retries por segundo.
   - Domain events relacionados (`workflow_application_onboarding_*`).

2. **ApplicationEnvironmentProvisioning**
   - Success Rate del workflow.
   - Duración p95 del workflow.
   - Retries por segundo.
   - Domain events relacionados (`workflow_appenv_provisioning_*`).

Este dashboard se centra en operar los workflows clave descritos en `docs/workflows/overview.md`.

## Evolución futura

- Añadir dashboards específicos por dominio (equipos, aplicaciones, secretos).
- Añadir anotaciones de despliegue y cambios de configuración.
- Integrar alertas basadas en estas mismas consultas.
