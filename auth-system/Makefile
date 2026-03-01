# Makefile — Auth System developer workflow
# Usage: make <target>
# All targets assume you are in the monorepo root.

BINARY_NAME   := auth-api
BUILD_DIR     := ./bin
GO_CMD        := go
DOCKER_IMAGE  := auth-api
BACKEND_DIR   := ./backend
MIGRATE_CMD   := $(GO_CMD) run $(BACKEND_DIR)/cmd/migrate/main.go

# Load .env.local if it exists (for local development convenience).
# In CI, environment variables are set by the pipeline.
ifneq (,$(wildcard .env.local))
  include .env.local
  export
endif

.PHONY: all dev build clean \
        migrate migrate-global migrate-tenant migrate-all-tenants \
        test test-unit test-integration test-isolation test-coverage \
        lint vet fmt \
        docker-build docker-push \
        seed help

# Default target
all: lint test build

# ---------------------------------------------------------------------------
# Development
# ---------------------------------------------------------------------------

## dev: Start all Docker services and run the API with live output
dev: docker-up
	@echo "Starting auth API..."
	cd $(BACKEND_DIR) && $(GO_CMD) run ./cmd/api

## docker-up: Start all local development services (postgres, redis, vault, mailhog)
docker-up:
	docker compose up -d
	@echo "Waiting for services to be healthy..."
	@docker compose ps

## docker-down: Stop all local development services (preserves volumes)
docker-down:
	docker compose down

## docker-reset: Stop all services and DELETE all data volumes (full reset)
docker-reset:
	docker compose down -v
	@echo "All volumes deleted. Run 'make dev' to start fresh."

# ---------------------------------------------------------------------------
# Database Migrations
# ---------------------------------------------------------------------------

## migrate: Run global migrations + all tenant schema migrations (up)
migrate: migrate-global migrate-all-tenants

## migrate-global: Apply pending migrations to the global (public) schema
migrate-global:
	@echo "Running global schema migrations..."
	cd $(BACKEND_DIR) && $(MIGRATE_CMD) -scope=global -direction=up

## migrate-all-tenants: Apply pending migrations to all active tenant schemas
migrate-all-tenants:
	@echo "Running all tenant schema migrations..."
	cd $(BACKEND_DIR) && $(MIGRATE_CMD) -scope=all -direction=up

## migrate-tenant TENANT=<slug>: Apply pending migrations to a specific tenant schema
## Example: make migrate-tenant TENANT=acme-corp
migrate-tenant:
ifndef TENANT
	$(error TENANT is not set. Usage: make migrate-tenant TENANT=acme-corp)
endif
	@echo "Running migrations for tenant: $(TENANT)"
	cd $(BACKEND_DIR) && $(MIGRATE_CMD) -scope=tenant -tenant=$(TENANT) -direction=up

## migrate-down TENANT=<slug>: Roll back one migration for a specific tenant
migrate-down:
ifndef TENANT
	$(error TENANT is not set. Usage: make migrate-down TENANT=acme-corp)
endif
	cd $(BACKEND_DIR) && $(MIGRATE_CMD) -scope=tenant -tenant=$(TENANT) -direction=down -steps=1

## migrate-create NAME=<name> SCOPE=<global|tenant>: Create a new migration file pair
migrate-create:
ifndef NAME
	$(error NAME is not set. Usage: make migrate-create NAME=add_user_phone SCOPE=tenant)
endif
ifndef SCOPE
	$(error SCOPE is not set. Usage: make migrate-create NAME=add_user_phone SCOPE=tenant)
endif
	@TIMESTAMP=$$(date +%06d) ; \
	  FILENAME="$${TIMESTAMP}_$(NAME)" ; \
	  touch $(BACKEND_DIR)/migrations/$(SCOPE)/$${FILENAME}.up.sql ; \
	  touch $(BACKEND_DIR)/migrations/$(SCOPE)/$${FILENAME}.down.sql ; \
	  echo "Created: migrations/$(SCOPE)/$${FILENAME}.up.sql" ; \
	  echo "Created: migrations/$(SCOPE)/$${FILENAME}.down.sql"

# ---------------------------------------------------------------------------
# Testing
# ---------------------------------------------------------------------------

## test: Run all tests (unit + integration + isolation)
test: test-unit test-integration test-isolation

## test-unit: Run unit tests only (no external services required)
test-unit:
	@echo "Running unit tests..."
	cd $(BACKEND_DIR) && $(GO_CMD) test \
	  -race \
	  -count=1 \
	  -timeout=60s \
	  -short \
	  ./internal/... ./pkg/...

## test-integration: Run integration tests (requires running Docker services)
test-integration:
	@echo "Running integration tests..."
	cd $(BACKEND_DIR) && $(GO_CMD) test \
	  -race \
	  -count=1 \
	  -timeout=120s \
	  -v \
	  ./test/integration/...

## test-isolation: Run cross-tenant isolation suite only (CI-blocking per US-08b)
## Any cross-tenant data leak in this suite = immediate failure.
test-isolation:
	@echo "Running cross-tenant isolation test suite..."
	cd $(BACKEND_DIR) && $(GO_CMD) test \
	  -race \
	  -count=1 \
	  -timeout=180s \
	  -v \
	  -run TestCrossTenant \
	  ./test/isolation/...

## test-coverage: Run tests with coverage report (fails if < 80%)
test-coverage:
	@echo "Running tests with coverage..."
	cd $(BACKEND_DIR) && $(GO_CMD) test \
	  -race \
	  -count=1 \
	  -timeout=120s \
	  -coverprofile=coverage.out \
	  ./internal/... ./pkg/...
	$(GO_CMD) tool cover -func=$(BACKEND_DIR)/coverage.out | tail -1
	@COVERAGE=$$(cd $(BACKEND_DIR) && $(GO_CMD) tool cover -func=coverage.out | tail -1 | awk '{print $$3}' | tr -d '%') ; \
	  echo "Coverage: $${COVERAGE}%" ; \
	  if [ "$$(echo "$${COVERAGE} < 80" | bc -l)" = "1" ] ; then \
	    echo "FAIL: Coverage $${COVERAGE}% is below the 80% threshold." ; \
	    exit 1 ; \
	  fi

# ---------------------------------------------------------------------------
# Code Quality
# ---------------------------------------------------------------------------

## lint: Run golangci-lint with project configuration
lint:
	@echo "Running golangci-lint..."
	cd $(BACKEND_DIR) && golangci-lint run ./...

## vet: Run go vet
vet:
	cd $(BACKEND_DIR) && $(GO_CMD) vet ./...

## fmt: Format all Go source files
fmt:
	cd $(BACKEND_DIR) && $(GO_CMD) fmt ./...

## check: Run all code quality checks (fmt + vet + lint)
check: fmt vet lint

# ---------------------------------------------------------------------------
# Build
# ---------------------------------------------------------------------------

## build: Build the production binary
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	cd $(BACKEND_DIR) && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
	  $(GO_CMD) build \
	  -ldflags="-w -s -X main.version=$(shell git describe --tags --always --dirty)" \
	  -o ../$(BUILD_DIR)/$(BINARY_NAME) \
	  ./cmd/api
	@echo "Binary: $(BUILD_DIR)/$(BINARY_NAME)"

## docker-build: Build the Docker image
docker-build:
	@echo "Building Docker image: $(DOCKER_IMAGE):local"
	docker build \
	  --build-arg VERSION=$(shell git describe --tags --always --dirty) \
	  -t $(DOCKER_IMAGE):local \
	  -t $(DOCKER_IMAGE):$(shell git rev-parse --short HEAD) \
	  $(BACKEND_DIR)

## docker-push: Push image to registry (set REGISTRY env var)
docker-push: docker-build
ifndef REGISTRY
	$(error REGISTRY is not set. Usage: REGISTRY=registry.example.com make docker-push)
endif
	docker tag $(DOCKER_IMAGE):local $(REGISTRY)/$(DOCKER_IMAGE):$(shell git rev-parse --short HEAD)
	docker push $(REGISTRY)/$(DOCKER_IMAGE):$(shell git rev-parse --short HEAD)

# ---------------------------------------------------------------------------
# Seeding
# ---------------------------------------------------------------------------

## seed: Create a test super-admin user and a test tenant ("dev-corp")
## Prints credentials to stdout — store them immediately.
seed:
	@echo "Seeding test data..."
	cd $(BACKEND_DIR) && $(GO_CMD) run ./scripts/seed.go

# ---------------------------------------------------------------------------
# Cleanup
# ---------------------------------------------------------------------------

## clean: Remove build artifacts
clean:
	rm -rf $(BUILD_DIR)
	rm -f $(BACKEND_DIR)/coverage.out
	@echo "Clean complete."

# ---------------------------------------------------------------------------
# Help
# ---------------------------------------------------------------------------

## help: Show this help message
help:
	@echo "Auth System — Available Makefile targets:"
	@echo ""
	@grep -E '^## ' Makefile | sed 's/## /  /' | column -t -s ':'
