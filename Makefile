# ============================================================================
# Systems Escape Rooms — Makefile
# Deterministic command surface for humans, agents, and CI.
# Rule: if an agent needs to run a command, it MUST exist here.
# ============================================================================

.PHONY: ci lint test-unit test-integration test-e2e build fmt gqlgen \
        dev-up dev-down dev-reset dev-logs \
        run-graphql run-engine-a run-engine-b run-judge-dispatcher \
        run-bundle-proxy run-artifact-proxy run-all \
        migrate-up migrate-down migrate-create \
        roomctl-validate roomctl-build \
        ui-install ui-dev ui-lint ui-test ui-build ui-codegen ui-fmt \
        help

# Default target
.DEFAULT_GOAL := help

# ── Configuration ──────────────────────────────────────────────────────────
GO         := go
GOTEST     := $(GO) test
GOBUILD    := $(GO) build
DOCKER_COMPOSE := docker compose -f infra/docker-compose.yaml

# UI (web/)
PNPM       := pnpm
UI_DIR     := web

# Load local env if it exists (for make run-* targets)
ifneq (,$(wildcard .env.local))
  include .env.local
  export
endif

# ── CI Gate (the one command that matters) ─────────────────────────────────
ci: lint test-unit test-integration build ui-lint ui-test ui-build ## Run full CI gate (backend + frontend)
	@echo "CI gate passed (backend + frontend)."

# ── Lint & Format ──────────────────────────────────────────────────────────
lint: ## Run all linters (go vet, staticcheck, format check)
	$(GO) vet ./...
	@if command -v staticcheck >/dev/null 2>&1; then \
		staticcheck ./...; \
	else \
		echo "SKIP: staticcheck not installed. Install with: go install honnef.co/go/tools/cmd/staticcheck@latest"; \
	fi
	@if command -v gofumpt >/dev/null 2>&1; then \
		OUTPUT=$$(gofumpt -d .); \
		if [ -n "$$OUTPUT" ]; then \
			echo "$$OUTPUT"; \
			echo "Run 'make fmt' to fix formatting"; \
			exit 1; \
		fi; \
	else \
		echo "SKIP: gofumpt not installed. Install with: go install mvdan.cc/gofumpt@latest"; \
	fi
	@# TODO(M0): Add custom architecture linter (layering rules)
	@echo "Lint passed."

fmt: ## Auto-fix formatting
	@if command -v gofumpt >/dev/null 2>&1; then \
		gofumpt -w .; \
		echo "Formatted."; \
	else \
		echo "gofumpt not installed. Install with:"; \
		echo "  go install mvdan.cc/gofumpt@latest"; \
	fi

gqlgen: ## Regenerate GraphQL code from schema
	$(GO) run github.com/99designs/gqlgen generate

# ── Tests ──────────────────────────────────────────────────────────────────
test-unit: ## Run unit tests (no external deps, <2 min target)
	$(GOTEST) -short -count=1 -race ./...
	@echo "Unit tests passed."

test-integration: ## Run integration tests (requires docker-compose services)
	$(GOTEST) -count=1 -race -run Integration ./...
	@echo "Integration tests passed."

test-e2e: ## Run E2E tests (requires full stack running)
	$(GOTEST) -count=1 -race -run E2E ./...
	@echo "E2E tests passed."

# ── Build ──────────────────────────────────────────────────────────────────
build: ## Build all service binaries
	@mkdir -p bin
	$(GOBUILD) -o bin/graphql-bff       ./cmd/graphql-bff
	$(GOBUILD) -o bin/engine-a          ./cmd/engine-a
	$(GOBUILD) -o bin/engine-b          ./cmd/engine-b-orchestrator
	$(GOBUILD) -o bin/judge-dispatcher  ./cmd/judge-dispatcher
	$(GOBUILD) -o bin/bundle-proxy      ./cmd/bundle-proxy
	$(GOBUILD) -o bin/artifact-proxy    ./cmd/artifact-proxy
	$(GOBUILD) -o bin/roomctl           ./cmd/roomctl
	$(GOBUILD) -o bin/migrate           ./cmd/migrate
	@echo "Build succeeded. Binaries in ./bin/"

# ── Local Infrastructure ───────────────────────────────────────────────────
dev-up: ## Start local infra (Postgres, Kafka, Redis, MinIO)
	$(DOCKER_COMPOSE) up -d
	@echo "Waiting for services..."
	@sleep 5
	@echo "Infra running. Postgres:5432 Kafka:9092 Redis:6379 MinIO:9000"

dev-down: ## Stop local infra
	$(DOCKER_COMPOSE) down

dev-reset: ## Stop infra and destroy all data volumes
	$(DOCKER_COMPOSE) down -v
	@echo "Volumes destroyed. Run 'make dev-up && make migrate-up' to restart."

dev-logs: ## Tail logs from all infra services
	$(DOCKER_COMPOSE) logs -f

# ── Migrations ─────────────────────────────────────────────────────────────
MIGRATE := $(GO) run ./cmd/migrate

migrate-up: ## Apply all pending Postgres migrations
	$(MIGRATE) up

migrate-down: ## Roll back the last migration
	$(MIGRATE) down 1

migrate-create: ## Create a new migration (usage: make migrate-create NAME=add_foo)
ifndef NAME
	$(error NAME is required. Usage: make migrate-create NAME=add_submissions_table)
endif
	$(MIGRATE) create $(NAME)

# ── Run Services (local dev) ───────────────────────────────────────────────
run-graphql: ## Run GraphQL BFF on :8080
	$(GO) run ./cmd/graphql-bff

run-engine-a: ## Run Engine A service on :8081
	$(GO) run ./cmd/engine-a

run-engine-b: ## Run Engine B Orchestrator on :8082
	$(GO) run ./cmd/engine-b-orchestrator

run-judge-dispatcher: ## Run Judge Dispatcher
	$(GO) run ./cmd/judge-dispatcher

run-bundle-proxy: ## Run Bundle Proxy on :8083
	$(GO) run ./cmd/bundle-proxy

run-artifact-proxy: ## Run Artifact Proxy on :8084
	$(GO) run ./cmd/artifact-proxy

run-all: ## Run all services (requires goreman or overmind)
	@if command -v goreman >/dev/null 2>&1; then \
		goreman start; \
	elif command -v overmind >/dev/null 2>&1; then \
		overmind start; \
	else \
		echo "Install goreman or overmind for multi-service runner."; \
		echo "   Or run each service in a separate terminal with make run-<service>"; \
		exit 1; \
	fi

# ── Room Content ───────────────────────────────────────────────────────────
roomctl-validate: ## Validate room content (schema + leak check)
ifdef ROOM
	$(GO) run ./cmd/roomctl validate --room $(ROOM)
else
	$(GO) run ./cmd/roomctl validate --all
endif
	@echo "Room validation passed."

roomctl-build: ## Build room bundles
ifdef ROOM
	$(GO) run ./cmd/roomctl build --room $(ROOM)
else
	$(GO) run ./cmd/roomctl build --all
endif
	@echo "Room bundles built."

# ── Web UI (web/) ──────────────────────────────────────────────────────
ui-install: ## Install web UI dependencies (pnpm install)
	@if [ -f $(UI_DIR)/package.json ]; then \
		cd $(UI_DIR) && $(PNPM) install --frozen-lockfile; \
		echo "UI dependencies installed."; \
	else \
		echo "SKIP: $(UI_DIR)/package.json not found. UI not set up yet."; \
	fi

ui-dev: ## Start Vite dev server on :3000
	@if [ -f $(UI_DIR)/package.json ]; then \
		cd $(UI_DIR) && $(PNPM) run dev; \
	else \
		echo "SKIP: $(UI_DIR)/package.json not found. UI not set up yet."; \
	fi

ui-lint: ## Run ESLint + TypeScript type-check on web/
	@if [ -f $(UI_DIR)/package.json ]; then \
		cd $(UI_DIR) && $(PNPM) run lint; \
		echo "UI lint passed."; \
	else \
		echo "SKIP: $(UI_DIR)/package.json not found. UI not set up yet."; \
	fi

ui-test: ## Run Vitest (unit + component tests) for web/
	@if [ -f $(UI_DIR)/package.json ]; then \
		cd $(UI_DIR) && $(PNPM) run test -- --run; \
		echo "UI tests passed."; \
	else \
		echo "SKIP: $(UI_DIR)/package.json not found. UI not set up yet."; \
	fi

ui-build: ## Build Vite production bundle
	@if [ -f $(UI_DIR)/package.json ]; then \
		cd $(UI_DIR) && $(PNPM) run build; \
		echo "UI production build succeeded."; \
	else \
		echo "SKIP: $(UI_DIR)/package.json not found. UI not set up yet."; \
	fi

ui-codegen: ## Generate TypeScript types from GraphQL schema
	@if [ -f $(UI_DIR)/package.json ]; then \
		cd $(UI_DIR) && $(PNPM) run codegen; \
		echo "GraphQL types regenerated in web/src/lib/graphql/types.ts"; \
	else \
		echo "SKIP: $(UI_DIR)/package.json not found. UI not set up yet."; \
	fi

ui-fmt: ## Auto-fix UI formatting (Prettier + ESLint --fix)
	@if [ -f $(UI_DIR)/package.json ]; then \
		cd $(UI_DIR) && $(PNPM) run format; \
		echo "UI formatted."; \
	else \
		echo "SKIP: $(UI_DIR)/package.json not found. UI not set up yet."; \
	fi

# ── Help ───────────────────────────────────────────────────────────────────
help: ## Show this help
	@echo "Systems Escape Rooms — Available targets:"
	@echo ""
	@grep -hE '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-22s\033[0m %s\n", $$1, $$2}'
	@echo ""
	@echo "CI gate (run before every PR):  make ci"
