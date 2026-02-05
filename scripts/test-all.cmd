@echo off
REM Ejecuta todos los tests dentro de contenedores Docker
cd /d "%~dp0.."
docker compose -f infra\docker-compose.yml up --build --abort-on-container-exit control-plane-api-tests workflow-engine-tests execution-workers-tests
