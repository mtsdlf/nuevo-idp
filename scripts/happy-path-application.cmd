@echo off
setlocal ENABLEDELAYEDEXPANSION

set BASE_URL=http://localhost:8080

echo === Happy path: Team + Application + Environments + Repos + GitOps ===

echo [1/10] Crear Team
curl -s -X POST "%BASE_URL%/commands/teams" ^
  -H "Content-Type: application/json" ^
  -d "{\"id\":\"team-1\",\"name\":\"Platform Team\"}"
echo.

echo [2/10] Crear Application
curl -s -X POST "%BASE_URL%/commands/applications" ^
  -H "Content-Type: application/json" ^
  -d "{\"id\":\"app-1\",\"name\":\"Sample App\",\"teamId\":\"team-1\"}"
echo.

echo [3/10] Aprobar Application
curl -s -X POST "%BASE_URL%/commands/applications/approve" ^
  -H "Content-Type: application/json" ^
  -d "{\"id\":\"app-1\"}"
echo.

echo [4/10] Crear Environment dev
curl -s -X POST "%BASE_URL%/commands/environments" ^
  -H "Content-Type: application/json" ^
  -d "{\"id\":\"env-dev\",\"name\":\"Development\"}"
echo.

echo [5/10] Crear Environment prod
curl -s -X POST "%BASE_URL%/commands/environments" ^
  -H "Content-Type: application/json" ^
  -d "{\"id\":\"env-prod\",\"name\":\"Production\"}"
echo.

echo [6/10] Declarar ApplicationEnvironment dev
curl -s -X POST "%BASE_URL%/commands/application-environments" ^
  -H "Content-Type: application/json" ^
  -d "{\"id\":\"app-1-env-dev\",\"applicationId\":\"app-1\",\"environmentId\":\"env-dev\"}"
echo.

echo [7/10] Declarar ApplicationEnvironment prod
curl -s -X POST "%BASE_URL%/commands/application-environments" ^
  -H "Content-Type: application/json" ^
  -d "{\"id\":\"app-1-env-prod\",\"applicationId\":\"app-1\",\"environmentId\":\"env-prod\"}"
echo.

echo [8/10] Declarar CodeRepository
curl -s -X POST "%BASE_URL%/commands/code-repositories" ^
  -H "Content-Type: application/json" ^
  -d "{\"id\":\"code-app-1\",\"applicationId\":\"app-1\"}"
echo.

echo [9/10] Declarar DeploymentRepository
curl -s -X POST "%BASE_URL%/commands/deployment-repositories" ^
  -H "Content-Type: application/json" ^
  -d "{\"id\":\"dep-app-1\",\"applicationId\":\"app-1\",\"deploymentModel\":\"GitOpsPerApplication\"}"
echo.

echo [10/10] Declarar GitOpsIntegration
curl -s -X POST "%BASE_URL%/commands/gitops-integrations" ^
  -H "Content-Type: application/json" ^
  -d "{\"id\":\"gi-app-1\",\"applicationId\":\"app-1\",\"deploymentRepositoryId\":\"dep-app-1\"}"
echo.

echo --- Consultar estado final de la Application ---
curl -s "%BASE_URL%/queries/applications?id=app-1"
echo.

echo Listar ApplicationEnvironment dev
curl -s "%BASE_URL%/queries/application-environments?id=app-1-env-dev"
echo.

echo Happy path completado.
pause
endlocal
