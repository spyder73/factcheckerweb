SHELL := /bin/bash
COMPOSE := docker compose -f infra/docker-compose.dev.yml --env-file .env

.PHONY: help dev down logs ps build api web scraper psql redis-cli fmt vet lint test migrate

help:  ## Show this help.
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

dev:  ## Bring up all services (postgres, redis, api, web, scraper) via docker compose.
	$(COMPOSE) up --build

down:  ## Stop and remove compose services.
	$(COMPOSE) down

logs:  ## Tail compose logs.
	$(COMPOSE) logs -f --tail=100

ps:  ## List compose services.
	$(COMPOSE) ps

build:  ## Build all compose images.
	$(COMPOSE) build

api:  ## Run the API directly with `go run` (needs env vars in shell).
	cd apps/api && go run .

web:  ## Run the Vite dev server directly.
	cd apps/web && npm run dev

scraper:  ## Run the Instagram scraper directly.
	cd services/scrapers/instagram && python app.py

psql:  ## psql into the dev postgres container.
	$(COMPOSE) exec postgres psql -U $${POSTGRES_USER:-alethea} -d $${POSTGRES_DB:-alethea}

redis-cli:  ## redis-cli into the dev redis container.
	$(COMPOSE) exec redis redis-cli

fmt:  ## Format Go code.
	cd apps/api && gofmt -w .

vet:  ## go vet.
	cd apps/api && go vet ./...

lint:  ## Lint Go + JS.
	cd apps/api && go vet ./...
	cd apps/web && npm run lint

test:  ## Run all tests (none yet — Phase 1+).
	cd apps/api && go test ./...

migrate:  ## Run DB migrations (Phase 1+).
	@echo "TODO Phase 1: wire goose against infra/migrations/"
