# Arquitectura del IDP

Este documento resume, en formato narrativo, los acuerdos recogidos en `architecture.md`.

## Principios centrales

- **Workflow-centric control plane**: el estado de negocio vive en el control plane, y las ejecuciones largas se modelan como workflows durables.
- **Intent separado de ejecución**: la API recibe comandos de intención; la ejecución concreta (side-effects) ocurre en workers.
- **Dominio como fuente de verdad**: el estado de dominio es canónico; los workflows no mutan directamente el dominio.
- **Auditability first**: cada transición importante debe ser auditable vía eventos, métricas y trazas.

## Límites de servicio (service boundaries)

### control-plane-api

Responsabilidades principales:

- Recibir comandos HTTP.
- Validar reglas de dominio.
- Mutar estado de dominio.
- Emitir eventos de dominio.
- Servir consultas.

Restricciones:

- No llamar proveedores externos directamente.
- No ejecutar side-effects.
- No contener lógica de retries.

### workflow-engine

Responsabilidades principales:

- Orquestar procesos de larga duración (workflows Temporal).
- Persistir estado de workflows.
- Gestionar retries, timeouts, pausas y reanudaciones.

Patrones clave:

- Orquestación tipo saga.
- Ejecución determinista.

### execution-workers

Responsabilidades principales:

- Ejecutar side-effects.
- Integrar con sistemas externos (GitHub, Harbor, etc.).
- Asegurar idempotencia.
- Normalizar errores externos a errores de dominio.

Restricciones:

- No decidir estado de negocio.
- No mutar dominio directamente.
- No aceptar requests humanas (solo llamadas internas).

### Autenticación interna entre servicios

Las llamadas HTTP entre `workflow-engine`, `control-plane-api` y `execution-workers` usan un esquema de autenticación interna muy simple basado en un token compartido:

- Header: `X-Internal-Token`.
- Origen: leido desde la variable de entorno `INTERNAL_AUTH_TOKEN` en cada servicio.
- Comportamiento:
	- Los adapters HTTP del `workflow-engine` siempre envían el header si `INTERNAL_AUTH_TOKEN` está configurada.
	- Los handlers internos de `control-plane-api` y `execution-workers` exigen que el header coincida con su propio `INTERNAL_AUTH_TOKEN` cuando ésta está configurada.
	- Si `INTERNAL_AUTH_TOKEN` no está seteada, el enforcement se desactiva (modo dev/local), pero se recomienda definirla siempre en entornos compartidos (CI, staging, producción).

## Patrones arquitectónicos

- **Hexagonal architecture** en API y workers: separación clara entre dominio, aplicación, puertos y adapters.
- **Orchestration over choreography** en workflows: el engine orquesta pasos explícitos para favorecer auditabilidad.
- **CQRS pragmático**: comandos para mutar, queries para leer (sin complicar en exceso los modelos de lectura).

Para detalles más finos (nombres exactos, reglas adicionales), ver `architecture.md` en la raíz del repositorio.
