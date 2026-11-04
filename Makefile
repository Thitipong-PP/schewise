.PHONY: help setup dev-api dev-web fmt fmt-check lint test build check smoke docker-up docker-down

help: ## Show available commands
	@awk 'BEGIN {FS = ":.*## "} /^[a-zA-Z_-]+:.*## / {printf "%-16s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

setup: ## Install dependencies
	go mod download
	pnpm install --frozen-lockfile

dev-api: ## Run the Go API on 127.0.0.1:8080
	go run ./cmd/api

dev-web: ## Run Next.js on 127.0.0.1:3000
	pnpm dev:web

fmt: ## Format Go, web, and documentation files
	gofmt -w cmd internal
	pnpm format

fmt-check: ## Check formatting without changing files
	@test -z "$$(gofmt -l cmd internal)" || (gofmt -l cmd internal; exit 1)
	pnpm format:check

lint: ## Run Go vet and ESLint
	go vet ./...
	pnpm lint

test: ## Run Go tests with the race detector
	go test -race -count=1 ./...

build: ## Build the API and production web app
	go build -trimpath -o bin/schewise-api ./cmd/api
	pnpm build

check: fmt-check lint test ## Run the same source checks as CI
	pnpm typecheck

smoke: ## Check running production builds (ports 8080 and 3000 must be free)
	bash scripts/smoke.sh

docker-up: ## Build and start the local development containers
	docker compose up --build

docker-down: ## Stop the local development containers
	docker compose down
