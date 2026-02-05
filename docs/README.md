# Documentación del IDP

Este directorio agrupa documentación viva sobre la arquitectura del IDP, sus servicios, sus workflows, su observabilidad y las fitness functions que protegen sus decisiones de diseño.

- `architecture/`: versión narrativa de `architecture.md` (principios, límites de servicio, patrones).
- `services/`: documentación por servicio (`control-plane-api`, `workflow-engine`, `execution-workers`).
- `workflows/`: visión de alto nivel de los workflows clave (onboarding, provisión de entornos, rotación de secretos) y cómo se relacionan con métricas y eventos.
- `observability/`: cómo medimos, trazamos y visualizamos el comportamiento del sistema.
- `operations/`: cómo levantar y operar el stack en local y en CI.
- `testing/`: estrategia de testing y fitness functions.
- `fitness-functions/`: tests y chequeos automáticos que protegen decisiones arquitectónicas y de producto.

La intención es complementar `architecture.md` (visión declarativa) con guías operativas y acuerdos concretos que se validan de forma automática.
