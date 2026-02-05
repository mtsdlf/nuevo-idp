@echo off
REM Ejecuta golangci-lint dentro de contenedor Docker
cd /d "%~dp0.."
docker compose -f infra\docker-compose.yml up --build --abort-on-container-exit golangci-lint
