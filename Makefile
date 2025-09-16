# =========================
#  Project configuration
# =========================
BINARY := basketball-stats-service
CMD_DIR := ./cmd/server
MIGRATIONS_DIR := ./migrations/goose_sql

# Go options
GO ?= go
GOCMD := $(GO)
GOTEST := $(GO) test
GOMOD := $(GO) mod
GOBUILD := $(GO) build
GORUN := $(GO) run

# Colors for pretty output
GREEN := \033[1;32m
RED := \033[1;31m
YELLOW := \033[1;33m
NC := \033[0m

# =========================
#  Targets
# =========================

## Default target
.DEFAULT_GOAL := help

## Build the service binary
build:
	@echo "$(YELLOW)🚀 Building binary: $(BINARY)...$(NC)"
	@if $(GOBUILD) -o $(BINARY) $(CMD_DIR); then \
		echo "$(GREEN)✅ Build successful: $(BINARY)$(NC)"; \
	else \
		echo "$(RED)❌ Build failed$(NC)"; \
		exit 1; \
	fi

## Run the service (from sources, without building binary)
run:
	@echo "$(YELLOW)🏃 Running service...$(NC)"
	@tmpfile=$$(mktemp -t bss-run.XXXX); \
	set -a; \
	. .env 2>/dev/null || true; \
	set +a; \
	bash -c 'set -o pipefail; go run ./cmd/server | tee "$$0"' "$$tmpfile"; \
	code=$$?; \
	# If non-zero but graceful termination detected in logs, treat as success
	if [ "$$code" -ne 0 ] && grep -q "Server exited" "$$tmpfile"; then \
		code=0; \
	fi; \
	rm -f "$$tmpfile"; \
	if [ "$$code" -eq 0 ] || [ "$$code" -eq 130 ] || [ "$$code" -eq 143 ]; then \
		echo "$(GREEN)✅ Server stopped gracefully (exit $$code)$(NC)"; \
		exit 0; \
	else \
		echo "$(RED)❌ Server exited with code $$code$(NC)"; \
		exit $$code; \
	fi

## Run tests
# Use -coverpkg=./... to instrument all packages even if tests live under ./test/ only.
# This gives us realistic coverage for internal/*, not 0.0%.
# coverage.out is used by CI to publish a summary or upload as artifact.
test:
	@echo "$(YELLOW)🧪 Running tests (with aggregated coverage)...$(NC)"
	@$(GOTEST) -race -covermode=atomic -coverpkg=./... -coverprofile=coverage.out ./...
	@$(GO) tool cover -func=coverage.out | tail -n 1

## Human friendly HTML coverage (optional)
test-html: test
	@$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)📊 Open coverage.html in browser for detailed report$(NC)"

## Run repository contract tests against a real Postgres
# Requires a running DB and proper env (can be provided via .env)
# Example: make test-contract
test-contract:
	@echo "$(YELLOW)🔗 Running repository contract tests...$(NC)"
	@set -a; \
	. .env 2>/dev/null || true; \
	set +a; \
	CONTRACT_TESTS=1 $(GO) test ./test/repository -run PostgresContract -race -count=1 -v

## Check formatting
fmt:
	@echo "$(YELLOW)🎨 Checking formatting...$(NC)"
	@test -z "$$($(GO) fmt ./...)" || (echo "$(RED)❌ Code not formatted$(NC)"; exit 1)

## Run go vet
vet:
	@echo "$(YELLOW)🔍 Running go vet...$(NC)"
	@$(GO) vet ./...

## Lint with golangci-lint
lint:
	@echo "$(YELLOW)🧹 Running golangci-lint...$(NC)"
	@golangci-lint run ./...

## Update dependencies
deps:
	@echo "$(YELLOW)📦 Tidying dependencies...$(NC)"
	@$(GOMOD) tidy
	@$(GOMOD) verify

# =========================
#  Database (Goose)
# =========================

define LOAD_ENV_AND_DBURL
	set -a; \
	. .env 2>/dev/null || true; \
	set +a; \
	if [ -z "$$DATABASE_URL" ]; then \
		USER_VAL="$$APP_POSTGRES_USER"; \
		[ -z "$$USER_VAL" ] && USER_VAL="$$POSTGRES_USER"; \
		[ -z "$$USER_VAL" ] && USER_VAL="$$DB_USER"; \
		[ -z "$$USER_VAL" ] && USER_VAL="postgres"; \
		PASS_VAL="$$APP_POSTGRES_PASSWORD"; \
		[ -z "$$PASS_VAL" ] && PASS_VAL="$$POSTGRES_PASSWORD"; \
		[ -z "$$PASS_VAL" ] && PASS_VAL="$$DB_PASSWORD"; \
		[ -z "$$PASS_VAL" ] && PASS_VAL="postgres"; \
		HOST_VAL="$$APP_POSTGRES_HOST"; \
		[ -z "$$HOST_VAL" ] && HOST_VAL="$$POSTGRES_HOST"; \
		[ -z "$$HOST_VAL" ] && HOST_VAL="localhost"; \
		PORT_VAL="$$APP_POSTGRES_PORT"; \
		[ -z "$$PORT_VAL" ] && PORT_VAL="$$POSTGRES_PORT"; \
		[ -z "$$PORT_VAL" ] && PORT_VAL="5432"; \
		DB_VAL="$$APP_POSTGRES_DB"; \
		[ -z "$$DB_VAL" ] && DB_VAL="$$POSTGRES_DB"; \
		[ -z "$$DB_VAL" ] && DB_VAL="$$DB_NAME"; \
		[ -z "$$DB_VAL" ] && DB_VAL="basketball"; \
		SSLMODE_VAL="$$APP_POSTGRES_SSLMODE"; \
		[ -z "$$SSLMODE_VAL" ] && SSLMODE_VAL="$$POSTGRES_SSLMODE"; \
		[ -z "$$SSLMODE_VAL" ] && SSLMODE_VAL="disable"; \
		export DATABASE_URL="postgres://$${USER_VAL}:$${PASS_VAL}@$${HOST_VAL}:$${PORT_VAL}/$${DB_VAL}?sslmode=$${SSLMODE_VAL}"; \
	fi
endef

## Run database migrations (up)
migrate-up:
	@echo "$(YELLOW)⬆️  Applying migrations...$(NC)"
	@$(LOAD_ENV_AND_DBURL); goose -dir $(MIGRATIONS_DIR) postgres "$$DATABASE_URL" up

## Rollback last migration (down by 1)
migrate-down:
	@echo "$(YELLOW)⬇️  Rolling back last migration...$(NC)"
	@$(LOAD_ENV_AND_DBURL); goose -dir $(MIGRATIONS_DIR) postgres "$$DATABASE_URL" down

## Check migration status
migrate-status:
	@$(LOAD_ENV_AND_DBURL); goose -dir $(MIGRATIONS_DIR) postgres "$$DATABASE_URL" status

## Print resolved DATABASE_URL for debugging
migrate-dsn:
	@$(LOAD_ENV_AND_DBURL); echo "DATABASE_URL=$$DATABASE_URL"

## Create new migration file
migrate-create:
	@if [ -z "$(name)" ]; then \
		echo "$(RED)❌ Please provide a migration name, e.g.: make migrate-create name=add_users_table$(NC)"; \
		exit 1; \
	fi
	@goose -dir $(MIGRATIONS_DIR) create $(name) sql

# =========================
#  Docker
# =========================

## Start Docker services
docker-up:
	@echo "$(YELLOW)🐳 Starting Docker containers...$(NC)"
	@docker-compose up -d

## Stop Docker services
docker-down:
	@echo "$(YELLOW)🛑 Stopping Docker containers...$(NC)"
	@docker-compose down

## Remove Docker volumes
docker-clean:
	@echo "$(RED)💣 Removing containers & volumes...$(NC)"
	@docker-compose down -v --remove-orphans

# =========================
#  Helpers
# =========================

## Show help (list targets)
help:
	@echo "$(GREEN)Available commands:$(NC)"
	@grep -E '^##' Makefile | sed -e 's/## //'

# =========================
#  CI / QA
# =========================

## Run full CI pipeline locally (fmt + vet + lint + test)
ci: fmt vet lint test
	@echo "$(GREEN)✅ CI checks passed$(NC)"

# Declare phony targets (directories with same names exist: e.g. test/). Without this, make thinks they are up to date.
.PHONY: build run test test-html test-contract fmt vet lint deps migrate-up migrate-down migrate-status migrate-dsn migrate-create docker-up docker-down docker-clean help ci
