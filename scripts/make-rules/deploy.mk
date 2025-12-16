# ==============================================================================
# Makefile helper functions for deployment
# ==============================================================================

.PHONY: deploy.run
deploy.run: ## Start all services using docker-compose.
	docker-compose -f deploy/docker-compose.yaml up -d

.PHONY: deploy.down
deploy.down: ## Stop all services using docker-compose.
	docker-compose -f deploy/docker-compose.yaml down

.PHONY: deploy.infra
deploy.infra: ## Start infrastructure services (MySQL, Redis) only.
	docker-compose -f deploy/docker-compose.yaml up -d mysql redis

.PHONY: deploy.clean
deploy.clean: ## Stop services and remove volumes.
	docker-compose -f deploy/docker-compose.yaml down -v
