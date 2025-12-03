# Smart Contract Event Indexer

> A high-performance blockchain event indexer built with Go microservices, designed to monitor smart contract events, parse and store them in PostgreSQL, and expose GraphQL/REST APIs for fast queries.

[![Go Version](https://img.shields.io/badge/Go-1.21+-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](CONTRIBUTING.md)

## 📋 Table of Contents

- [Overview](#overview)
- [Features](#features)
- [Architecture](#architecture)
- [Tech Stack](#tech-stack)
- [Quick Start](#quick-start)
- [Development](#development)
- [Project Structure](#project-structure)
- [Configuration](#configuration)
- [Testing](#testing)
- [Deployment](#deployment)
- [Documentation](#documentation)
- [Contributing](#contributing)
- [License](#license)

## 🎯 Overview

Smart Contract Event Indexer solves the problem of slow and expensive direct blockchain queries by:
- **Real-time indexing** of smart contract events
- **Fast querying** through PostgreSQL with optimized indexes
- **Flexible APIs** via GraphQL and REST
- **Reliable data** with chain reorganization handling

### Core Value Propositions

- 🚀 **Performance**: Event indexing delay < 90s (6-block confirmation)
- ⚡ **Speed**: API response time P95 < 200ms
- 📊 **Scalability**: Handles 1000+ events/second
- 🔄 **Reliability**: 99.99% data accuracy with reorg handling
- 🎯 **Flexibility**: Configurable confirmation strategies (1, 6, or 12 blocks)

## ✨ Features

### Core Functionality

- ✅ **Multi-Contract Monitoring** - Track multiple smart contracts simultaneously
- ✅ **Event Parsing** - Automatic ABI-based event parsing
- ✅ **Chain Reorg Handling** - Detect and handle blockchain reorganizations
- ✅ **Historical Backfill** - Index historical events from any block
- ✅ **GraphQL API** - Flexible query interface with pagination
- ✅ **REST API** - Traditional HTTP endpoints
- ✅ **Real-time Updates** - Low-latency event indexing
- ✅ **Caching Layer** - Redis-based caching for hot queries

### Advanced Features

- 🎛️ **Configurable Confirmations** - Choose between realtime (1 block), balanced (6 blocks), or safe (12 blocks)
- 📈 **Contract Statistics** - Built-in analytics and metrics
- 🔍 **JSONB Queries** - Flexible event argument filtering
- 🔄 **Automatic Reconnection** - Resilient RPC connection management
- 📊 **Prometheus Metrics** - Production-ready monitoring
- 🐳 **Docker Ready** - Complete containerization support

## 🏗️ Architecture

### Microservices Overview

```
┌─────────────────────────────────────────┐
│  Client (DApp / Dashboard / Analytics)  │
└──────────────┬──────────────────────────┘
               │ GraphQL/REST
┌──────────────▼──────────────────────────┐
│         API Gateway (Port 8000)         │
│  - GraphQL (gqlgen) / REST (Gin)       │
│  - Auth & Rate Limiting                 │
└──────────────┬──────────────────────────┘
               │ gRPC
┌──────────────▼──────────────────────────┐
│  Query Service (8081) | Admin (8082)   │
│  - Caching (Redis)    | - Management   │
│  - Aggregations       | - Monitoring   │
└──────────────┬──────────────────────────┘
               │
┌──────────────▼──────────────────────────┐
│      Indexer Service (Port 8080)        │
│  - Blockchain Monitoring (WebSocket)    │
│  - Event Parsing (go-ethereum)          │
│  - Reorg Handling                       │
└──────────────┬──────────────────────────┘
               │
         ┌─────┴──────┐
         ▼            ▼
    PostgreSQL    Blockchain Node
    + Redis       (Geth/Infura/Ganache)
```

### Service Responsibilities

| Service | Port | Responsibility |
|---------|------|----------------|
| **Indexer Service** | 8080 | Blockchain monitoring, event parsing, storage |
| **API Gateway** | 8000 | Public API endpoints, authentication, rate limiting |
| **Query Service** | 8081 | Query optimization, caching, aggregations |
| **Admin Service** | 8082 | Contract management, monitoring, backfill jobs |

## 🛠️ Tech Stack

### Backend
- **Language**: Go 1.21+
- **Database**: PostgreSQL 15 (JSONB for flexible event args)
- **Cache**: Redis 7 (query caching, session management)
- **RPC**: gRPC (inter-service communication)
- **API**: GraphQL (gqlgen) + REST (Gin)

### Blockchain
- **Client**: go-ethereum (geth)
- **Testnet**: Ganache (local development)
- **Production**: Alchemy/Infura (Ethereum mainnet)

### Infrastructure
- **Containers**: Docker + Docker Compose
- **Orchestration**: Kubernetes (K8s)
- **CI/CD**: GitHub Actions / GitLab CI
- **Monitoring**: Prometheus + Grafana

## 🚀 Quick Start

### Prerequisites

- **Docker & Docker Compose** - [Install Docker](https://docs.docker.com/get-docker/)
- **Go 1.21+** - [Install Go](https://golang.org/doc/install)
- **Make** - Usually pre-installed on Unix systems

### Installation

```bash
# Clone the repository
git clone https://github.com/yourusername/smart-contract-event-indexer.git
cd smart-contract-event-indexer

# Complete setup (installs deps, starts Docker, runs migrations)
make setup

# Verify all services are running
make health-check
```

That's it! Your development environment is ready. 🎉

### Access Services

| Service | URL | Description |
|---------|-----|-------------|
| **Landing Page** | `http://localhost:8000` | Project overview & API links |
| **GraphQL Playground** | `http://localhost:8000/playground` | Interactive API testing |
| **GraphQL API** | `http://localhost:8000/graphql` | GraphQL endpoint |
| **REST API** | `http://localhost:8000/api/v1/*` | REST endpoints |
| **PostgreSQL** | `localhost:5432` | Database (user: indexer) |
| **Redis** | `localhost:6379` | Cache |
| **Ganache RPC** | `http://localhost:8545` | Local blockchain |
| **Adminer** | `http://localhost:8080` | Database UI |

## 💻 Development

### Common Commands

```bash
# Start development environment
make dev-up

# Build all services
make build

# Run tests
make test

# Run linter
make lint

# Format code
make fmt

# View logs
make docker-logs

# Stop environment
make dev-down

# Database shell
make db-shell

# Redis CLI
make redis-cli
```

### Running Individual Services

```bash
# Run indexer service locally
make run-indexer

# Run API gateway
make run-api

# Run query service
make run-query

# Run admin service
make run-admin
```

### Database Migrations

```bash
# Run migrations
make migrate-up

# Rollback migrations
make migrate-down

# Create new migration
make migrate-create NAME=add_new_feature

# Force migration version
make migrate-force VERSION=1
```

### Generating Code

```bash
# Generate gRPC code from proto files
make proto-gen
```

## 📁 Project Structure

```
mono-repo/
├── services/               # Microservices
│   ├── indexer-service/   # Blockchain event indexing
│   ├── api-gateway/       # GraphQL/REST API
│   │   └── web/           # Embedded frontend (Landing Page)
│   ├── query-service/     # Query optimization
│   └── admin-service/     # Admin & management
├── shared/                 # Shared code
│   ├── models/            # Data models
│   ├── proto/             # gRPC definitions
│   ├── config/            # Configuration
│   ├── utils/             # Utilities
│   └── database/          # Database helpers
├── infrastructure/         # Infrastructure as code
│   ├── docker/            # Dockerfiles
│   ├── k8s/               # Kubernetes manifests
│   └── terraform/         # Terraform configs
├── migrations/            # Database migrations
├── graphql/               # GraphQL schemas
├── tests/                 # Test suites
│   └── loadtest/          # Load testing tools
├── scripts/               # Utility scripts
├── docs/                  # Documentation
├── docker-compose.yml     # Local development
├── Makefile              # Build automation
└── go.work               # Go workspace
```

## ⚙️ Configuration

### Environment Variables

Create a `.env` file based on `.env.example`:

```bash
# Database
DATABASE_URL=postgres://indexer:indexer_password@localhost:5432/event_indexer?sslmode=disable

# Redis
REDIS_URL=redis://localhost:6379

# Blockchain RPC
RPC_ENDPOINT=http://localhost:8545
# RPC_ENDPOINT=https://eth-mainnet.g.alchemy.com/v2/YOUR_KEY

# Indexer Settings
INDEXER_BATCH_SIZE=100
INDEXER_DEFAULT_CONFIRM_BLOCKS=6  # balanced mode
INDEXER_POLL_INTERVAL=6s

# Confirmation Strategies
# - realtime: 1 block (~12s delay)
# - balanced: 6 blocks (~72s delay) - RECOMMENDED
# - safe: 12 blocks (~144s delay)

# Logging
LOG_LEVEL=info
LOG_FORMAT=json  # or "text" for development

# Environment
ENVIRONMENT=development
```

### Configuration Strategies

**Confirmation Blocks** determine how many blocks to wait before considering an event "final":

| Strategy | Blocks | Delay | Accuracy | Use Case |
|----------|--------|-------|----------|----------|
| Realtime | 1 | ~12s | ~99% | Demos, non-critical apps |
| Balanced | 6 | ~72s | ~99.99% | Most production apps (RECOMMENDED) |
| Safe | 12 | ~144s | ~99.9999% | Financial apps, auditing |

## 🧪 Testing

```bash
# Run all tests
make test

# Run with coverage
make test-coverage

# Run unit tests only
make test-unit

# Run integration tests
make test-integration

# View coverage report
open coverage.html
```

### Load Testing

```bash
# Standard load test (100 concurrent, 1000 requests)
make loadtest

# Quick test (50 concurrent, 500 requests)
make loadtest-quick

# Stress test (200 concurrent, 30s duration)
make loadtest-stress
```

**Performance Targets**: P95 < 200ms, Success Rate ≥ 99%

### Test Structure

- **Unit Tests**: Test individual functions and components
- **Integration Tests**: Test service interactions
- **Load Tests**: Performance and concurrency testing
- **E2E Tests**: Test complete workflows with Docker services

## 🚢 Deployment

### Docker

```bash
# Build Docker images
make docker-build

# Start all services
make docker-up

# View container status
make docker-ps

# View logs
make docker-logs

# Stop all services
make docker-down
```

### Kubernetes

```bash
# Apply configurations
kubectl apply -f infrastructure/k8s/

# Check status
kubectl get pods -n event-indexer

# View logs
kubectl logs -f deployment/indexer-service -n event-indexer
```

### Production Considerations

1. **RPC Provider**: Use reliable providers (Alchemy, Infura)
2. **Database**: Scale PostgreSQL with read replicas
3. **Redis**: Use Redis Cluster for high availability
4. **Monitoring**: Set up Prometheus + Grafana
5. **Logging**: Centralize with ELK or Loki
6. **Backups**: Regular database backups
7. **Security**: Enable SSL, use secrets management

## 📚 Documentation

### Project Documentation

- [Progress Dashboard](docs/PROGRESS.md) - Current project status and metrics
- [Quick Reference](docs/QUICK_REFERENCE.md) - Developer quick reference
- [Architecture Overview](docs/smart_contract_event_indexer_architecture.md)
- [Product Requirements](docs/smart_contract_event_indexer_prd.md)
- [Implementation Plan](docs/smart_contract_event_indexer_plan.md)

### Architecture Documentation

- [System Architecture](docs/architecture/diagrams/system-architecture.md) - High-level system design
- [Architecture Decisions](docs/architecture/decisions/) - ADRs for major decisions
- [Git Workflow](docs/development/GIT_WORKFLOW.md) - Development workflow guidelines

### API Documentation

- **Landing Page**: `http://localhost:8000` - Project overview with live status
- **GraphQL Playground**: `http://localhost:8000/playground` - Interactive API testing
- REST API: See [API Documentation](docs/api/rest-endpoints.md)

### Frontend Deployment (Vercel)

The landing page can be deployed separately to Vercel for free hosting:

```bash
cd services/api-gateway/web
# Edit config.js to set API_URL to your backend
vercel deploy
```

The frontend is also embedded in the Go binary for single-deployment scenarios.

## 🤝 Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details.

### Development Workflow

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'feat: add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

### Commit Convention

We use [Conventional Commits](https://www.conventionalcommits.org/):

- `feat:` - New feature
- `fix:` - Bug fix
- `docs:` - Documentation changes
- `refactor:` - Code refactoring
- `test:` - Test additions or changes
- `chore:` - Build process or auxiliary tool changes

## 📊 Project Status

**Current Phase**: Phase 4 - Query Service ✅ Complete

**Overall Progress**: 90% (18/20 tasks)

### Roadmap

- [x] **Phase 1**: Infrastructure setup ✅
- [x] **Phase 2**: Indexer Service core ✅
- [x] **Phase 3**: API layer ✅
- [x] **Phase 4**: Query Service & optimization ✅
- [ ] **Phase 5**: Deployment & production

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- [go-ethereum](https://github.com/ethereum/go-ethereum) - Ethereum client library
- [gqlgen](https://github.com/99designs/gqlgen) - GraphQL code generation
- [Gin](https://github.com/gin-gonic/gin) - HTTP web framework
- [PostgreSQL](https://www.postgresql.org/) - Database system
- [Redis](https://redis.io/) - In-memory data store

## 📧 Contact

For questions or support, please open an issue on GitHub.

---

**Built with ❤️ by the Smart Contract Event Indexer Team**

*Happy Indexing! 🚀*

