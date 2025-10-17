<!-- 4e80c50f-5b91-475b-b486-b0690619c775 c0d3268d-5ec8-40ef-af6e-df4a57df2e95 -->
# Phase 1: Project Infrastructure Setup

## Overview

Establish the complete mono-repo foundation with Go workspaces, shared modules, and Docker development environment. By the end of this phase, you'll have a fully functional development environment ready for building the Indexer Service.

## 1. Mono-repo Initialization

### 1.1 Project Structure Setup

- Create the mono-repo directory structure following the architecture:
  - `services/` - microservices (indexer, api-gateway, query, admin)
  - `shared/` - shared code (proto, models, utils, config)
  - `infrastructure/` - Docker, K8s, Terraform configs
  - `migrations/` - database migration scripts
  - `graphql/` - GraphQL schema definitions
  - `docs/development/` - development logs and feature documentation

### 1.2 Go Workspace Configuration

- Create `go.work` file to manage multiple Go modules
- Initialize Go modules for each service and shared package
- Set up proper module dependencies

### 1.3 Build System

- Create comprehensive `Makefile` with commands for:
  - Building all services (`make build`)
  - Running tests (`make test`, `make test-coverage`)
  - Code generation (`make generate`)
  - Linting and formatting (`make lint`, `make fmt`)
  - Database migrations (`make migrate-up`, `make migrate-down`)
  - Running individual services (`make run-indexer`, etc.)
  - Docker operations (`make docker-build`, `make docker-up`)

### 1.4 Project Configuration Files

- `.gitignore` - ignore build artifacts, vendor, IDE files
- `.editorconfig` - consistent coding style
- `.golangci.yml` - linter configuration
- `README.md` - project introduction and quick start guide

## 2. Shared Modules Development

### 2.1 Data Models (`shared/models/`)

Define core data structures:

- `Contract` - smart contract metadata
- `Event` - blockchain event data
- `EventArg` - event argument details
- `IndexerState` - indexing progress tracking
- Common types (Address, Hash, BigInt handling)

### 2.2 gRPC Protocol Definitions (`shared/proto/`)

Define service interfaces:

- `query_service.proto` - event query operations
- `admin_service.proto` - contract management operations
- Generate Go code with protoc

### 2.3 Configuration Management (`shared/config/`)

- Config loader supporting environment variables and YAML
- Configuration structs for each service
- Validation logic for configuration values
- Default values and required field checks

### 2.4 Logging Framework (`shared/utils/logger.go`)

- Structured logging with levels (DEBUG, INFO, WARN, ERROR)
- JSON format for production, pretty format for development
- Context-aware logging with trace IDs
- Service name tagging

### 2.5 Error Handling (`shared/utils/errors.go`)

- Custom error types for different scenarios
- Error wrapping with context
- gRPC error code mapping
- Error logging middleware

### 2.6 Database Utilities (`shared/database/`)

- Connection pool management
- Transaction helpers
- Common query builders
- Health check functions

## 3. Docker Development Environment

### 3.1 Docker Compose Setup

Create `docker-compose.yml` with services:

- **PostgreSQL 15** - main database with persistent volume
- **Redis 7** - caching layer with persistence
- **Ganache/Hardhat** - local Ethereum test network
- **Adminer** - database management UI (optional)

Configuration requirements:

- Proper networking between services
- Volume mounts for data persistence
- Health checks for all services
- Environment variable management

### 3.2 Database Initialization

#### Schema Design (`migrations/001_initial_schema.sql`)

Tables to create:

- `contracts` - monitored smart contracts
  - Fields: id, address, abi, name, start_block, current_block, confirm_blocks
  - Indexes: unique on address
- `events` - indexed blockchain events
  - Fields: id, contract_address, event_name, block_number, transaction_hash, log_index, args (JSONB), timestamp
  - Indexes: composite on (contract_address, block_number), GIN on args
- `indexer_state` - tracking indexing progress
  - Fields: contract_address, last_indexed_block, updated_at
- `block_cache` - for reorg detection
  - Fields: block_number, block_hash, parent_hash, timestamp
  - Index: on block_number DESC (keep last 100)

#### Migration Tool Setup

- Integrate `golang-migrate` library
- Create migration management scripts
- Up/down migration commands in Makefile

### 3.3 Test Network Setup

- Configure Ganache with deterministic accounts
- OR set up Hardhat node with forked mainnet
- Document RPC endpoints and test accounts
- Create sample ERC20/ERC721 contracts for testing

### 3.4 Health Check Scripts

- Database connectivity check
- Redis connectivity check
- RPC node availability check
- Overall system health endpoint

## 4. Development Workflow Setup

### 4.1 Environment Variables

Create `.env.example` with:

```bash
# Database
DATABASE_URL=postgres://user:pass@localhost:5432/event_indexer

# Redis
REDIS_URL=redis://localhost:6379

# Blockchain RPC
RPC_ENDPOINT=http://localhost:8545

# Service Ports
INDEXER_SERVICE_PORT=8080
API_GATEWAY_PORT=8000
QUERY_SERVICE_PORT=8081
ADMIN_SERVICE_PORT=8082
```

### 4.2 Documentation Setup

Based on the documentation standards rule, create:

- `docs/development/features/` - feature development logs
- `docs/development/bugs/` - bug fix logs
- `docs/development/debug-sessions/` - debugging sessions
- Initial feature log: `docs/development/features/001-project-infrastructure.md`

## Acceptance Criteria

At the end of Phase 1, you should have:

- [ ] Complete mono-repo structure with all directories
- [ ] Go workspace configured with all modules
- [ ] Makefile with all essential commands working
- [ ] Shared models and utilities implemented
- [ ] gRPC proto files defined and generated
- [ ] Docker Compose environment starts successfully
- [ ] PostgreSQL database with initial schema
- [ ] Redis running and accessible
- [ ] Local test network (Ganache/Hardhat) running
- [ ] All services can connect to dependencies
- [ ] Documentation structure in place
- [ ] README with quick start instructions

## Key Files to Create

**Root Level:**

- `go.work`
- `Makefile`
- `docker-compose.yml`
- `.gitignore`
- `.editorconfig`
- `.golangci.yml`
- `README.md`
- `.env.example`

**Shared Package:**

- `shared/go.mod`
- `shared/models/*.go`
- `shared/proto/*.proto`
- `shared/config/config.go`
- `shared/utils/logger.go`
- `shared/utils/errors.go`
- `shared/database/db.go`

**Infrastructure:**

- `infrastructure/docker/Dockerfile.indexer`
- `migrations/001_initial_schema.up.sql`
- `migrations/001_initial_schema.down.sql`

**Documentation:**

- `docs/development/features/001-project-infrastructure.md`

## Next Steps After Phase 1

Once infrastructure is complete, you'll move to Phase 2 (Indexer Service Core):

1. Blockchain connection module with RPC management
2. Event parsing with ABI handling
3. Data persistence with batch operations
4. Chain reorganization detection and handling
5. Main indexer loop integration

## Success Validation

Run these commands to verify Phase 1 completion:

```bash
# Start environment
make dev-up

# Check all services are healthy
docker-compose ps

# Run database migration
make migrate-up

# Build all modules
make build

# Run linter
make lint

# Verify connection
# (Will implement actual health checks in services)
```

### To-dos

- [ ] Create complete mono-repo directory structure with services/, shared/, infrastructure/, migrations/, graphql/, and docs/ directories
- [ ] Set up Go workspace with go.work file and initialize go.mod for shared package and each service placeholder
- [ ] Create project configuration files: .gitignore, .editorconfig, .golangci.yml, .env.example
- [ ] Create comprehensive Makefile with build, test, lint, migration, docker, and service run commands
- [ ] Implement shared data models: Contract, Event, EventArg, IndexerState, and common types
- [ ] Define gRPC proto files for query_service and admin_service, generate Go code
- [ ] Implement configuration loader supporting ENV and YAML with validation
- [ ] Implement structured logging framework with levels and JSON/pretty formatting
- [ ] Implement error handling utilities with custom types and gRPC mapping
- [ ] Implement database utilities: connection pool, transaction helpers, health checks
- [ ] Create initial database migration with contracts, events, indexer_state, and block_cache tables
- [ ] Create docker-compose.yml with PostgreSQL, Redis, Ganache/Hardhat, and Adminer services
- [ ] Configure local Ethereum test network (Ganache or Hardhat) with deterministic accounts
- [ ] Implement health check scripts for database, Redis, and RPC node connectivity
- [ ] Set up documentation structure and create initial feature log for infrastructure setup
- [ ] Write comprehensive README.md with project overview, tech stack, and quick start guide
- [ ] Verify complete setup: start Docker environment, run migrations, test connections