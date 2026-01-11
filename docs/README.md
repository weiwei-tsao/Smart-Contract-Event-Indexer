# Smart Contract Event Indexer

A high-performance blockchain event indexing system built with Go microservices architecture. This project demonstrates Web3 development skills, microservices design patterns, and production-grade engineering practices.

## Table of Contents

1. [Project Overview](#project-overview)
2. [Architecture](#architecture)
3. [Technology Stack](#technology-stack)
4. [Database Schema](#database-schema)
5. [Data Flow](#data-flow)
6. [API Reference](#api-reference)
7. [Configuration](#configuration)
8. [Deployment](#deployment)
9. [Key Design Decisions](#key-design-decisions)
10. [Monitoring](#monitoring)

---

## Project Overview

### Purpose

The Smart Contract Event Indexer solves the problem of slow and expensive direct blockchain queries by:

- **Real-time indexing** of smart contract events with configurable confirmation strategies
- **Fast querying** via PostgreSQL optimized with JSONB and GIN indexes
- **Flexible APIs** supporting both GraphQL and REST endpoints
- **Reliable data handling** with chain reorganization (reorg) detection and recovery

### Core Value Proposition

| Metric | Target |
|--------|--------|
| Indexing Delay (Balanced Mode) | ~72 seconds |
| API Response P95 | < 200ms |
| Throughput | 1000+ events/second |
| Data Accuracy | 99.99% |
| Deployment Cost | $0-5/month |

### System Scope

**In Scope:**
- Monitoring and indexing blockchain events
- GraphQL/REST query APIs
- Data aggregation and statistics
- Historical data backfill
- System monitoring and alerting

**Out of Scope:**
- Smart contract write operations
- Blockchain node operations
- Frontend application development
- Complex user identity management

---

## Architecture

### Microservices Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                        Client Layer                              │
│     DApp Frontend  │  Analytics  │  Admin Dashboard              │
└─────────────┬──────┴──────┬──────┴────────┬─────────────────────┘
              │ GraphQL/REST │               │ HTTP
              └──────────────┴───────────────┘
                             │
┌────────────────────────────▼────────────────────────────────────┐
│                     API Gateway (8000)                           │
│  - GraphQL Server (gqlgen)    - REST API (Gin)                  │
│  - Authentication             - Rate Limiting                    │
└────────────┬─────────────────────────────────┬──────────────────┘
             │ gRPC                            │ gRPC
  ┌──────────▼──────────┐           ┌──────────▼─────────┐
  │  Query Service      │           │  Admin Service     │
  │      (8081)         │           │      (8082)        │
  │  - Cache Layer      │           │  - Management      │
  │  - Query Optimizer  │           │  - Backfill Jobs   │
  └──────────┬──────────┘           └──────────┬─────────┘
             │                                  │
┌────────────▼──────────────────────────────────▼─────────────────┐
│                    Indexer Service (8080)                        │
│  Event Listener → Parser → Validator → Storage                  │
│       │                                    │                     │
│       └────► Reorg Detector ◄──────────────┘                    │
└───────────┬─────────────────────────────────────────┬───────────┘
            │ WebSocket/HTTP                          │ SQL
            │                                         │
┌───────────▼──────────────┐          ┌───────────────▼───────────┐
│   Blockchain RPC Nodes   │          │      PostgreSQL 15        │
│  - Primary (Alchemy)     │          │  - events table           │
│  - Fallback nodes        │          │  - contracts table        │
└──────────────────────────┘          │  - indexer_state table    │
                                      └───────────────┬───────────┘
                                                      │
                                      ┌───────────────▼───────────┐
                                      │       Redis 7 Cache       │
                                      │  - Query Cache            │
                                      │  - Block State Cache      │
                                      └───────────────────────────┘
```

### Service Responsibilities

| Service | Port | Responsibility |
|---------|------|----------------|
| **Indexer Service** | 8080 | Blockchain monitoring, event parsing, data storage, reorg handling |
| **API Gateway** | 8000 | Public API endpoints, authentication, rate limiting, request routing |
| **Query Service** | 8081 | Query optimization, caching, data aggregation |
| **Admin Service** | 8082 | Contract management, backfill operations, system monitoring |

---

## Technology Stack

### Backend

| Component | Technology | Purpose |
|-----------|------------|---------|
| Language | Go 1.21+ | High performance, concurrency |
| Blockchain | go-ethereum | Ethereum interaction |
| GraphQL | gqlgen | Type-safe code generation |
| HTTP | Gin | REST API framework |
| IPC | gRPC | Service-to-service communication |

### Data & Caching

| Component | Technology | Purpose |
|-----------|------------|---------|
| Database | PostgreSQL 15 | Event storage, JSONB flexibility |
| Cache | Redis 7 | Query caching, task queue |
| Indexing | GIN indexes | Fast JSONB queries |

### Infrastructure

| Component | Technology | Purpose |
|-----------|------------|---------|
| Containers | Docker | Service containerization |
| Orchestration | Kubernetes | Production deployment |
| Monitoring | Prometheus + Grafana | Metrics and dashboards |
| Logging | Zap | Structured logging |

---

## Database Schema

### Core Tables

```sql
-- Monitored smart contracts
CREATE TABLE contracts (
    id SERIAL PRIMARY KEY,
    address VARCHAR(42) UNIQUE NOT NULL,
    name VARCHAR(100),
    abi JSONB NOT NULL,
    start_block BIGINT NOT NULL,
    current_block BIGINT DEFAULT 0,
    confirm_blocks INTEGER DEFAULT 6,  -- 1=realtime, 6=balanced, 12=safe
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Indexed blockchain events
CREATE TABLE events (
    id BIGSERIAL PRIMARY KEY,
    contract_id INTEGER REFERENCES contracts(id),
    contract_address VARCHAR(42) NOT NULL,
    event_name VARCHAR(100) NOT NULL,
    block_number BIGINT NOT NULL,
    block_hash VARCHAR(66),
    block_timestamp TIMESTAMP NOT NULL,
    transaction_hash VARCHAR(66) NOT NULL,
    transaction_index INTEGER NOT NULL,
    log_index INTEGER NOT NULL,
    args JSONB NOT NULL,
    raw_log JSONB,
    created_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(transaction_hash, log_index)
);

-- Indexing progress per contract
CREATE TABLE indexer_state (
    contract_address VARCHAR(42) PRIMARY KEY,
    last_indexed_block BIGINT NOT NULL,
    last_indexed_at TIMESTAMP DEFAULT NOW(),
    error_message TEXT
);

-- Block cache for reorg detection
CREATE TABLE block_cache (
    block_number BIGINT PRIMARY KEY,
    block_hash VARCHAR(66) NOT NULL,
    parent_hash VARCHAR(66) NOT NULL,
    cached_at TIMESTAMP DEFAULT NOW()
);
```

### Key Indexes

```sql
CREATE INDEX idx_events_contract ON events(contract_address, event_name);
CREATE INDEX idx_events_block ON events(block_number DESC);
CREATE INDEX idx_events_timestamp ON events(block_timestamp DESC);
CREATE INDEX idx_events_tx ON events(transaction_hash);
CREATE INDEX idx_events_args ON events USING GIN(args);
CREATE INDEX idx_events_contract_block ON events(contract_address, block_number DESC);
```

---

## Data Flow

### Event Indexing Pipeline

```
1. Block Monitor
   └─► Polls RPC for new blocks (every 6 seconds)

2. Event Retrieval
   └─► eth_getLogs(contractAddress, fromBlock, toBlock)
       └─► Respects confirmation blocks (1/6/12)

3. Event Parser
   └─► Decodes log parameters using contract ABI
   └─► Type conversion (BigNumber → String)
   └─► Extracts indexed + non-indexed parameters

4. Reorg Detector
   └─► Checks block hash consistency
   └─► Rolls back if reorg detected
   └─► Updates confirmed events

5. Batch Storage
   └─► Accumulates 100-500 events
   └─► Bulk inserts to PostgreSQL
   └─► Updates indexer_state
   └─► Invalidates related caches
```

### Query Execution Flow

```
1. Client Request
   └─► GraphQL/REST to API Gateway

2. API Gateway
   └─► Authentication
   └─► Rate limiting
   └─► Route to Query Service (gRPC)

3. Query Service
   └─► Check Redis cache
   └─► If miss: Query PostgreSQL
   └─► Cache result (TTL 30s)

4. Response
   └─► Return to client via API Gateway
```

---

## API Reference

### REST Endpoints

#### Health Check
```
GET /api/v1/health
```

#### Contract Management
```
GET    /api/v1/contracts                    # List all contracts
POST   /api/v1/contracts                    # Add new contract
GET    /api/v1/contracts/{address}          # Get contract details
DELETE /api/v1/contracts/{address}          # Remove contract
GET    /api/v1/contracts/{address}/stats    # Get contract statistics
```

#### Event Queries
```
GET /api/v1/events
    ?contract=0x...
    &event_name=Transfer
    &from_block=1000000
    &to_block=1001000
    &limit=50

GET /api/v1/events/tx/{txHash}              # Events by transaction
GET /api/v1/events/address/{address}        # Events by address
```

### GraphQL Schema

```graphql
type Query {
  events(
    filter: EventFilter
    pagination: PaginationInput
  ): EventConnection!

  eventsByTransaction(txHash: String!): [Event!]!
  eventsByAddress(address: String!, pagination: PaginationInput): EventConnection!

  contract(address: String!): Contract
  contracts(isActive: Boolean): [Contract!]!
  contractStats(address: String!): ContractStats!
}

type Mutation {
  addContract(input: AddContractInput!): AddContractPayload!
  removeContract(address: String!): RemoveContractPayload!
  triggerBackfill(address: String!, fromBlock: Int!, toBlock: Int!): BackfillPayload!
}

type Event {
  id: ID!
  contractAddress: String!
  eventName: String!
  blockNumber: Int!
  blockTimestamp: DateTime!
  transactionHash: String!
  args: [EventArg!]!
}

type EventConnection {
  edges: [EventEdge!]!
  pageInfo: PageInfo!
  totalCount: Int!
}
```

### Response Codes

| Code | Description |
|------|-------------|
| 200 | Success |
| 201 | Created |
| 400 | Bad Request |
| 404 | Not Found |
| 429 | Rate Limited |
| 500 | Server Error |

---

## Configuration

### Environment Variables

```bash
# Database
DATABASE_URL=postgres://user:pass@localhost:5432/indexer

# Cache
REDIS_URL=redis://localhost:6379

# Blockchain RPC
RPC_ENDPOINT=wss://eth-mainnet.g.alchemy.com/v2/YOUR_KEY
RPC_FALLBACK_1=https://rpc.ankr.com/eth

# Indexer Settings
INDEXER_BATCH_SIZE=100
INDEXER_DEFAULT_CONFIRM_BLOCKS=6
INDEXER_POLL_INTERVAL=6s

# Logging
LOG_LEVEL=info
LOG_FORMAT=json
```

### Confirmation Block Strategies

| Strategy | Blocks | Delay | Accuracy | Use Case |
|----------|--------|-------|----------|----------|
| **Realtime** | 1 | ~12s | ~99% | Demo, testing |
| **Balanced** (default) | 6 | ~72s | ~99.99% | Production apps |
| **Safe** | 12 | ~144s | ~99.9999% | Financial, audit |

---

## Deployment

### Local Development

```bash
# Start all services
docker-compose up -d

# Run database migrations
docker-compose run --rm migrate

# Check service health
curl http://localhost:8000/api/v1/health
```

### Production Options

#### Option 1: Railway.app (Recommended for Portfolio)

**Cost: $0-5/month**

```yaml
Services:
  - indexer-service
  - api-gateway
  - PostgreSQL (plugin)
  - Redis (plugin)

External (Free Tier):
  - Alchemy RPC: 300M CU/month
  - BetterUptime: Monitoring
```

#### Option 2: Kubernetes

```yaml
# Deployment
kubectl apply -f infrastructure/k8s/

# Components
- indexer-service: 2 replicas
- api-gateway: 3 replicas
- query-service: 3 replicas
- admin-service: 2 replicas
- HPA for auto-scaling
```

#### Option 3: Oracle Cloud Free Tier

**Cost: $0 (permanent)**

- VM.Standard.A1.Flex: 4 OCPUs, 24GB RAM
- Deploy via Docker Compose
- 200GB block storage

---

## Key Design Decisions

### ADR-001: Microservices Architecture

**Decision:** Use microservices instead of monolith.

**Rationale:**
- Independent scaling of query-heavy vs write-heavy services
- Fault isolation between indexing and API
- Technology flexibility per service

**Trade-offs:**
- Increased operational complexity
- Network latency between services
- Requires service discovery

### ADR-002: GraphQL as Primary API

**Decision:** GraphQL with REST fallback.

**Rationale:**
- Flexible queries reduce over-fetching
- Self-documenting schema
- Single endpoint for complex data

### ADR-003: PostgreSQL JSONB for Event Args

**Decision:** Store event arguments in JSONB with GIN indexes.

**Rationale:**
- Flexibility for dynamic event schemas
- GIN indexes provide good query performance
- Simpler than dedicated tables per event type

**Optimization Path:**
If P95 > 500ms, introduce dedicated `event_addresses` table.

### ADR-004: Configurable Confirmation Blocks

**Decision:** Allow per-contract confirmation strategy.

**Rationale:**
- Different apps have different latency/safety requirements
- Default 6 blocks balances speed and reliability
- User explicitly chooses the trade-off

---

## Monitoring

### Prometheus Metrics

```
# Indexer
indexer_lag_seconds              # Blocks behind chain head
indexer_events_processed_total   # Cumulative events indexed
indexer_rpc_calls_total          # RPC endpoint usage
indexer_reorg_detected_total     # Chain reorgs detected

# API
api_request_duration_seconds     # Response time histogram
api_requests_total               # Request count by endpoint

# Database
db_query_duration_seconds        # Query performance
db_connections_active            # Connection pool usage

# Cache
cache_hit_rate                   # Redis cache effectiveness
```

### Health Endpoints

| Service | Endpoint | Purpose |
|---------|----------|---------|
| API Gateway | `/api/v1/health` | Public health check |
| Query Service | `/health` | Internal health |
| Indexer | `/health` | Blockchain connection status |
| All Services | `/metrics` | Prometheus scrape |

### Alert Thresholds

| Metric | Warning | Critical |
|--------|---------|----------|
| Indexer Lag | > 2 min | > 5 min |
| API P95 | > 300ms | > 500ms |
| Error Rate | > 1% | > 5% |
| Cache Hit Rate | < 60% | < 40% |

---

## Project Structure

```
smart-contract-event-indexer/
├── services/
│   ├── indexer-service/     # Blockchain monitoring
│   ├── api-gateway/         # GraphQL/REST API
│   ├── query-service/       # Query optimization
│   └── admin-service/       # Management
├── shared/
│   ├── models/              # Data structures
│   ├── config/              # Configuration
│   ├── database/            # DB/Redis clients
│   └── proto/               # gRPC definitions
├── migrations/              # Database schemas
├── graphql/                 # GraphQL schema
├── infrastructure/          # Docker, K8s configs
├── tests/                   # Integration tests
├── Makefile                 # Build automation
└── docker-compose.yml       # Development environment
```

---

## Quick Commands

```bash
# Development
make dev-up              # Start local environment
make migrate-up          # Run migrations
make test                # Run all tests
make lint                # Run linters

# Build
make build               # Build all services
make docker-build        # Build Docker images

# Deployment
docker-compose up        # Local deployment
railway up               # Deploy to Railway
```

---

## License

MIT License - See LICENSE file for details.

---

**Version:** 1.0
**Last Updated:** 2026-01-10
