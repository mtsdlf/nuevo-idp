@echo off
setlocal ENABLEDELAYEDEXPANSION

set BASE_URL=http://localhost:8080

echo === Happy path: Secret + SecretBinding + Rotation ===

echo [1/6] Crear Team para el Secret
curl -s -X POST "%BASE_URL%/commands/teams" ^
  -H "Content-Type: application/json" ^
  -d "{\"id\":\"team-sec-1\",\"name\":\"Security Team\"}"
echo.

echo [2/6] Crear Secret
curl -s -X POST "%BASE_URL%/commands/secrets" ^
  -H "Content-Type: application/json" ^
  -d "{\"id\":\"secret-1\",\"ownerTeamId\":\"team-sec-1\",\"purpose\":\"sample-db-password\",\"sensitivity\":\"high\"}"
echo.

echo [3/6] Declarar SecretBinding apuntando al Team
curl -s -X POST "%BASE_URL%/commands/secret-bindings" ^
  -H "Content-Type: application/json" ^
  -d "{\"id\":\"sb-1\",\"secretId\":\"secret-1\",\"targetId\":\"team-sec-1\",\"targetType\":\"Team\"}"
echo.

echo [4/6] Iniciar rotación de Secret
curl -s -X POST "%BASE_URL%/commands/secrets/start-rotation" ^
  -H "Content-Type: application/json" ^
  -d "{\"id\":\"secret-1\"}"
echo.

echo [5/6] Completar rotación de Secret
curl -s -X POST "%BASE_URL%/commands/secrets/complete-rotation" ^
  -H "Content-Type: application/json" ^
  -d "{\"id\":\"secret-1\"}"
echo.

echo [6/6] Tocar /metrics para observabilidad
curl -s "%BASE_URL%/metrics" >NUL

echo Happy path de rotación de Secret completado (revisá métricas y eventos en Prometheus/Grafana).
pause
endlocal
