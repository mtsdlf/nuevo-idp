# platform

Módulo compartido con utilidades transversales para los servicios del IDP.

Paquetes principales:

- `observability`: inicialización de logger, métricas y helpers como `InstrumentHTTP`, `ObserveDomainEvent`, etc.
- `httpx`: helpers HTTP comunes (`WriteJSON`, `DecodeJSON`, `RequireMethod`, ...).
- `config`: lectura tipada de configuración y variables de entorno.
- `errors`: tipos y helpers de errores de dominio (Kind, código, mapeo a HTTP, etc.).
- `tracing`: inicialización de tracing con OpenTelemetry.

## Uso

Cada servicio (`control-plane-api`, `workflow-engine`, `execution-workers`) importa este módulo como `github.com/nuevo-idp/platform` en lugar de definir utilidades locales duplicadas.

Las reglas sobre su uso están descritas en [docs/architecture/overview.md](../docs/architecture/overview.md).
