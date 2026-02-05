# Levantar la observabilidad en local

Este documento resume cómo levantar el stack de observabilidad completo en local usando `infra/docker-compose.yml`.

## Requisitos

- Docker y Docker Compose instalados.
- Puerto libres: 3000 (Grafana), 9090 (Prometheus), 3200 (Tempo), 4318 (OTLP HTTP), 8080/8081/8082 (servicios IDP).

## Pasos

1. Desde el directorio `infra/`:

   ```bash
   docker compose up --build
   ```

2. Endpoints principales:

   - Control-plane API: http://localhost:8080
   - Workflow engine: http://localhost:8081
   - Execution workers: expone HTTP en 8082 (no suele tener UI propia).
   - Prometheus: http://localhost:9090
   - Grafana: http://localhost:3000 (usuario `admin` / contraseña `admin`).
   - Tempo (vía Grafana Explore, datasource `Tempo`).

3. Validar observabilidad básica:

- Visitar `/metrics` en cada servicio (`/metrics` en 8080, 8081, 8082) y comprobar que responde.
- Entrar a Grafana → dashboard `IDP Observability` y verificar que hay series para `http_requests_total`.
- Generar algunas requests (por ejemplo, curl contra control-plane-api) y comprobar que aparecen en Prometheus y que Tempo recibe trazas (via Explore → Tempo → `service.name="control-plane-api"`).

4. Ejecutar scripts de happy path

En Windows podés usar los scripts de ejemplo en el directorio `scripts/` para generar tráfico y estados de dominio representativos:

- `scripts\happy-path-application.cmd`: recorre el flujo feliz de creación de Team, Application, Environments, ApplicationEnvironments y GitOps (repositorios + integración) usando solo el control-plane-api.
- `scripts\happy-path-secret-rotation.cmd`: recorre un flujo feliz simplificado de creación de Secret, creación de SecretBinding y una rotación completa del Secret.

Ambos scripts asumen que el stack está levantado con `docker compose up` en `infra/` y que el control-plane-api está disponible en `http://localhost:8080`.

Con esto se obtiene un entorno local completo para experimentar con métricas, trazas y dashboards antes de desplegar en otros entornos.
