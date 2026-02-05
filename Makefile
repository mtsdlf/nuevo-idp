DOCKER_COMPOSE=cd infra && docker compose -f docker-compose.yml

.PHONY: test test-api test-workflow test-workers lint smoke ci

# Mantiene el comportamiento anterior: levanta todos los contenedores de tests
# a la vez y corta cuando uno termina.
test:
	$(DOCKER_COMPOSE) up --build --abort-on-container-exit control-plane-api-tests workflow-engine-tests execution-workers-tests

test-api:
	$(DOCKER_COMPOSE) run --rm control-plane-api-tests

test-workflow:
	$(DOCKER_COMPOSE) run --rm workflow-engine-tests

test-workers:
	$(DOCKER_COMPOSE) run --rm execution-workers-tests

lint:
	$(DOCKER_COMPOSE) run --rm golangci-lint

smoke:
	$(DOCKER_COMPOSE) run --rm smoke-tests

# Pipeline completa: tests de los tres servicios, lint y smoke secuenciales
ci:
	$(DOCKER_COMPOSE) run --rm control-plane-api-tests
	$(DOCKER_COMPOSE) run --rm workflow-engine-tests
	$(DOCKER_COMPOSE) run --rm execution-workers-tests
	$(DOCKER_COMPOSE) run --rm golangci-lint
	$(DOCKER_COMPOSE) run --rm smoke-tests
