.PHONY: help init setup dev-api dev-web fmt fmt-check lint test integration-test build check smoke auth-smoke docker-up docker-down local-ca migrate migrate-status

help: ## Show available commands
	@awk 'BEGIN {FS = ":.*## "} /^[a-zA-Z_-]+:.*## / {printf "%-16s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

setup: ## Install dependencies
	go mod download
	pnpm install --frozen-lockfile

init: ## Generate unique local credentials in .env without overwriting it
	node scripts/init-dev.mjs

dev-api: ## Run the Go API on 127.0.0.1:8080
	node scripts/with-env.mjs go run ./cmd/api

dev-web: ## Run Next.js on 127.0.0.1:3000
	node scripts/with-env.mjs pnpm dev:web

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

integration-test: ## Run authentication tests in isolated PostgreSQL schemas
	SCHEWISE_INTEGRATION_TESTS=1 node scripts/with-env.mjs go test -race -count=1 ./...

build: ## Build the API and production web app
	go build -trimpath -o bin/schewise-api ./cmd/api
	go build -trimpath -o bin/schewise-migrate ./cmd/migrate
	pnpm build

check: fmt-check lint test ## Run the same source checks as CI
	pnpm test:toolchain
	pnpm typecheck

smoke: ## Check production integration using the app database (ports 8080/3000 free)
	node scripts/with-env.mjs bash scripts/smoke.sh

auth-smoke: ## Check the running HTTPS login flow using installation credentials
	NODE_EXTRA_CA_CERTS=.local/root.crt node scripts/with-env.mjs node scripts/auth-smoke.mjs

docker-up: ## Build and start the local development containers
	docker compose up --build

docker-down: ## Stop the local development containers
	docker compose down

local-ca: ## Export the local HTTPS public CA certificate for client trust
	bash scripts/local-ca.sh

migrate: ## Apply application database migrations (also applied at API startup)
	node scripts/with-env.mjs go run ./cmd/migrate up

migrate-status: ## Show the current application database migration version
	node scripts/with-env.mjs go run ./cmd/migrate status
