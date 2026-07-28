.PHONY: build up down logs clean test help

help: ## Show this help
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "\033[36m%-10s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build all Docker images
	docker compose build --no-cache

up: ## Start all services (requires .env)
	cp .env.example .env 2>/dev/null || true
	docker compose up -d

down: ## Stop all services
	docker compose down

logs: ## Show logs from all services
	docker compose logs -f

clean: ## Stop services and remove volumes
	docker compose down -v
	rm -f .env

test: ## Run Go tests
	go test -race ./...

test-integration: ## Run integration tests (requires services running)
	go test -race -tags=integration ./...
