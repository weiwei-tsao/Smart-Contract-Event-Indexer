.PHONY: help build test test-coverage lint fmt clean docker-build docker-up docker-down dev-up dev-down migrate-up migrate-down migrate-create proto-gen run-indexer run-api run-query run-admin deps install-tools

# Default target
.DEFAULT_GOAL := help

# Colors for output
BLUE := \033[0;34m
GREEN := \033[0;32m
YELLOW := \033[0;33m
NC := \033[0m # No Color

help: ## Show this help message
	@echo '$(BLUE)Smart Contract Event Indexer - Makefile Commands$(NC)'
	@echo ''
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "$(GREEN)%-20s$(NC) %s\n", $$1, $$2}'

# Development
deps: ## Download Go module dependencies
	@echo "$(BLUE)Downloading dependencies...$(NC)"
	@go work sync
	@cd shared && go mod download
	@cd services/indexer-service && go mod download
	@cd services/api-gateway && go mod download
	@cd services/query-service && go mod download
	@cd services/admin-service && go mod download
	@echo "$(GREEN)Dependencies downloaded$(NC)"

install-tools: ## Install development tools
	@echo "$(BLUE)Installing development tools...$(NC)"
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	@go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	@go install github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	@echo "$(GREEN)Tools installed$(NC)"

# Building
build: ## Build all services
	@echo "$(BLUE)Building all services...$(NC)"
	@cd services/indexer-service && go build -o ../../bin/indexer-service ./cmd/main.go
	@cd services/api-gateway && go build -o ../../bin/api-gateway ./cmd/main.go
	@cd services/query-service && go build -o ../../bin/query-service ./cmd/main.go
	@cd services/admin-service && go build -o ../../bin/admin-service ./cmd/main.go
	@echo "$(GREEN)Build complete$(NC)"

build-indexer: ## Build indexer service
	@echo "$(BLUE)Building indexer service...$(NC)"
	@cd services/indexer-service && go build -o ../../bin/indexer-service ./cmd/main.go
	@echo "$(GREEN)Indexer service built$(NC)"

build-api: ## Build API gateway
	@echo "$(BLUE)Building API gateway...$(NC)"
	@cd services/api-gateway && go build -o ../../bin/api-gateway ./cmd/main.go
	@echo "$(GREEN)API gateway built$(NC)"

build-query: ## Build query service
	@echo "$(BLUE)Building query service...$(NC)"
	@cd services/query-service && go build -o ../../bin/query-service ./cmd/main.go
	@echo "$(GREEN)Query service built$(NC)"

build-admin: ## Build admin service
	@echo "$(BLUE)Building admin service...$(NC)"
	@cd services/admin-service && go build -o ../../bin/admin-service ./cmd/main.go
	@echo "$(GREEN)Admin service built$(NC)"

# Testing
test: ## Run all tests
	@echo "$(BLUE)Running tests...$(NC)"
	@go test -v ./...

test-coverage: ## Run tests with coverage
	@echo "$(BLUE)Running tests with coverage...$(NC)"
	@go test -v -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)Coverage report generated: coverage.html$(NC)"

test-unit: ## Run unit tests only
	@echo "$(BLUE)Running unit tests...$(NC)"
	@go test -v -short ./...

test-integration: ## Run integration tests
	@echo "$(BLUE)Running integration tests...$(NC)"
	@go test -v -run Integration ./...

# Code Quality
lint: ## Run linter
	@echo "$(BLUE)Running linter...$(NC)"
	@golangci-lint run --config .golangci.yml ./...

fmt: ## Format code
	@echo "$(BLUE)Formatting code...$(NC)"
	@go fmt ./...
	@goimports -w .

clean: ## Clean build artifacts
	@echo "$(BLUE)Cleaning build artifacts...$(NC)"
	@rm -rf bin/
	@rm -rf coverage.out coverage.html
	@find . -name '*.test' -delete
	@find . -name '*.out' -delete
	@echo "$(GREEN)Cleaned$(NC)"

# Docker
docker-build: ## Build Docker images
	@echo "$(BLUE)Building Docker images...$(NC)"
	@docker build -f infrastructure/docker/Dockerfile.indexer -t event-indexer/indexer-service:latest .
	@docker build -f infrastructure/docker/Dockerfile.api-gateway -t event-indexer/api-gateway:latest .
	@echo "$(GREEN)Docker images built$(NC)"

docker-up: ## Start all Docker services
	@echo "$(BLUE)Starting Docker services...$(NC)"
	@docker-compose up -d
	@echo "$(GREEN)Docker services started$(NC)"

docker-down: ## Stop all Docker services
	@echo "$(BLUE)Stopping Docker services...$(NC)"
	@docker-compose down
	@echo "$(GREEN)Docker services stopped$(NC)"

docker-logs: ## Show Docker logs
	@docker-compose logs -f

docker-ps: ## Show Docker container status
	@docker-compose ps

# Development Environment
dev-up: ## Start development environment
	@echo "$(BLUE)Starting development environment...$(NC)"
	@docker-compose up -d postgres redis ganache adminer
	@echo "$(YELLOW)Waiting for services to be ready...$(NC)"
	@sleep 5
	@echo "$(GREEN)Development environment ready!$(NC)"
	@echo ""
	@echo "$(BLUE)Available services:$(NC)"
	@echo "  PostgreSQL: localhost:5432 (user: indexer, pass: indexer_password)"
	@echo "  Redis: localhost:6379"
	@echo "  Ganache: http://localhost:8545"
	@echo "  Adminer: http://localhost:8080"

dev-down: ## Stop development environment
	@echo "$(BLUE)Stopping development environment...$(NC)"
	@docker-compose down
	@echo "$(GREEN)Development environment stopped$(NC)"

dev-restart: dev-down dev-up ## Restart development environment

# Database Migrations
migrate-up: ## Run database migrations up
	@echo "$(BLUE)Running migrations up...$(NC)"
	@docker-compose --profile tools run --rm migrate up
	@echo "$(GREEN)Migrations applied$(NC)"

migrate-down: ## Run database migrations down
	@echo "$(BLUE)Running migrations down...$(NC)"
	@docker-compose --profile tools run --rm migrate down
	@echo "$(GREEN)Migrations rolled back$(NC)"

migrate-create: ## Create a new migration (usage: make migrate-create NAME=migration_name)
	@if [ -z "$(NAME)" ]; then \
		echo "$(YELLOW)Usage: make migrate-create NAME=migration_name$(NC)"; \
		exit 1; \
	fi
	@echo "$(BLUE)Creating migration: $(NAME)...$(NC)"
	@migrate create -ext sql -dir migrations -seq $(NAME)
	@echo "$(GREEN)Migration created$(NC)"

migrate-force: ## Force migration version (usage: make migrate-force VERSION=1)
	@if [ -z "$(VERSION)" ]; then \
		echo "$(YELLOW)Usage: make migrate-force VERSION=1$(NC)"; \
		exit 1; \
	fi
	@echo "$(BLUE)Forcing migration version to $(VERSION)...$(NC)"
	@docker-compose --profile tools run --rm migrate force $(VERSION)
	@echo "$(GREEN)Migration version forced$(NC)"

# Proto Generation
proto-gen: ## Generate gRPC code from proto files
	@echo "$(BLUE)Generating gRPC code...$(NC)"
	@cd shared/proto && protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		*.proto
	@echo "$(GREEN)gRPC code generated$(NC)"

# Running Services
run-indexer: ## Run indexer service locally
	@echo "$(BLUE)Running indexer service...$(NC)"
	@cd services/indexer-service && go run ./cmd/main.go

run-api: ## Run API gateway locally
	@echo "$(BLUE)Running API gateway...$(NC)"
	@cd services/api-gateway && go run ./cmd/main.go

run-query: ## Run query service locally
	@echo "$(BLUE)Running query service...$(NC)"
	@cd services/query-service && go run ./cmd/main.go

run-admin: ## Run admin service locally
	@echo "$(BLUE)Running admin service...$(NC)"
	@cd services/admin-service && go run ./cmd/main.go

# Health Checks
health-check: ## Check health of all services
	@echo "$(BLUE)Checking service health...$(NC)"
	@echo "$(YELLOW)PostgreSQL:$(NC)"
	@docker-compose exec -T postgres pg_isready -U indexer || echo "  $(YELLOW)Not ready$(NC)"
	@echo "$(YELLOW)Redis:$(NC)"
	@docker-compose exec -T redis redis-cli ping || echo "  $(YELLOW)Not ready$(NC)"
	@echo "$(YELLOW)Ganache:$(NC)"
	@curl -s http://localhost:8545 > /dev/null && echo "  $(GREEN)Ready$(NC)" || echo "  $(YELLOW)Not ready$(NC)"

# Database Utils
db-shell: ## Connect to PostgreSQL shell
	@docker-compose exec postgres psql -U indexer -d event_indexer

db-reset: ## Reset database (drop and recreate)
	@echo "$(YELLOW)Warning: This will delete all data!$(NC)"
	@read -p "Are you sure? [y/N] " -n 1 -r; \
	echo; \
	if [[ $$REPLY =~ ^[Yy]$$ ]]; then \
		echo "$(BLUE)Resetting database...$(NC)"; \
		docker-compose --profile tools run --rm migrate drop -f; \
		$(MAKE) migrate-up; \
		echo "$(GREEN)Database reset complete$(NC)"; \
	fi

redis-cli: ## Connect to Redis CLI
	@docker-compose exec redis redis-cli

# Utility
logs-postgres: ## Show PostgreSQL logs
	@docker-compose logs -f postgres

logs-redis: ## Show Redis logs
	@docker-compose logs -f redis

logs-ganache: ## Show Ganache logs
	@docker-compose logs -f ganache

ps: docker-ps ## Alias for docker-ps

status: health-check ## Alias for health-check

# Complete Setup
setup: ## Complete project setup
	@echo "$(BLUE)Setting up project...$(NC)"
	@$(MAKE) deps
	@$(MAKE) proto-gen
	@$(MAKE) dev-up
	@$(MAKE) migrate-up
	@echo "$(GREEN)Setup complete!$(NC)"
	@echo ""
	@echo "$(BLUE)Next steps:$(NC)"
	@echo "  1. Run 'make build' to build all services"
	@echo "  2. Run 'make run-indexer' to start the indexer"
	@echo "  3. Run 'make test' to run tests"

