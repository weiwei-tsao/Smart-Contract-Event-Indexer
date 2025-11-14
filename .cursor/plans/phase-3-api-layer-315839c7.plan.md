<!-- 315839c7-54f9-4d27-98c0-b4c8b68db580 ca78fd65-1e00-4577-bc7b-5b5914f80070 -->
# Phase 3 - API Layer Development Plan

## Overview

Build the API Layer to expose indexed blockchain events through GraphQL/REST APIs. This includes implementing the API Gateway, Query Service with caching, and Admin Service for system management.

**Duration**: 1-2 weeks

**Prerequisites**: Phase 2 (Indexer Service) complete ✅

**Dependencies**: PostgreSQL, Redis, gRPC proto definitions

---

## Architecture Context

```
Client → API Gateway (GraphQL/REST) → Query Service (gRPC) → Database
                ↓                            ↓
           Admin Service (gRPC)          Redis Cache
```

**Services to Build**:

1. **API Gateway** (Port 8000): Public-facing GraphQL/REST API
2. **Query Service** (Port 8081): Optimized queries with caching
3. **Admin Service** (Port 8082): Contract management and backfill

---

## Implementation Tasks

### Task 1: GraphQL Schema Design & Code Generation

**Location**: `graphql/schema.graphql`

Design the complete GraphQL schema following the PRD specifications:

**Core Types**:

- `Event` - Blockchain event with all metadata
- `EventArg` - Key-value pair for event arguments
- `Contract` - Monitored contract configuration
- `EventConnection` - Relay-style cursor pagination
- `ContractStats` - Aggregated statistics

**Queries**:

- `events()` - Main event query with filtering
- `eventsByTransaction()` - Events by tx hash
- `eventsByAddress()` - Events involving an address
- `contract()` - Single contract details
- `contractStats()` - Aggregated statistics

**Mutations**:

- `addContract()` - Add new contract (idempotent)
- `removeContract()` - Stop indexing a contract
- `triggerBackfill()` - Historical data backfill

**Key Requirements**:

- Use custom scalars: `DateTime`, `BigInt`, `Address`
- Implement cursor-based pagination
- All mutations must be idempotent
- BigInt values returned as strings (avoid precision loss)

**Steps**:

1. Create `graphql/schema.graphql` with complete type definitions
2. Configure gqlgen in `services/api-gateway/gqlgen.yml`
3. Run code generation: `go run github.com/99designs/gqlgen generate`
4. Review generated resolver stubs

**Files Created**:

- `graphql/schema.graphql`
- `services/api-gateway/gqlgen.yml`
- `services/api-gateway/graph/model/models_gen.go` (generated)
- `services/api-gateway/graph/schema.resolvers.go` (generated stubs)

---

### Task 2: gRPC Service Definitions

**Location**: `shared/proto/`

Define gRPC interfaces for service communication:

**query_service.proto**:

```protobuf
service QueryService {
  rpc GetEvents(EventQuery) returns (EventResponse);
  rpc GetEventsByAddress(AddressQuery) returns (EventResponse);
  rpc GetEventsByTransaction(TransactionQuery) returns (EventResponse);
  rpc GetContractStats(StatsQuery) returns (StatsResponse);
}
```

**admin_service.proto**:

```protobuf
service AdminService {
  rpc AddContract(AddContractRequest) returns (AddContractResponse);
  rpc RemoveContract(RemoveContractRequest) returns (RemoveContractResponse);
  rpc TriggerBackfill(BackfillRequest) returns (BackfillResponse);
  rpc GetSystemStatus(Empty) returns (SystemStatusResponse);
}
```

**Steps**:

1. Design complete protobuf messages
2. Configure buf.yaml for code generation
3. Generate Go code: `buf generate`
4. Verify generated files compile

**Files Updated**:

- `shared/proto/query_service.proto`
- `shared/proto/admin_service.proto`
- `shared/proto/gen/` (generated code)

---

### Task 3: Query Service Implementation

**Location**: `services/query-service/`

Implement the Query Service with intelligent caching and query optimization.

**Core Components**:

**3.1 gRPC Server Setup**

- Implement `QueryServiceServer` interface
- Configure server with interceptors (logging, metrics, recovery)
- Health check endpoint

**3.2 Query Builder**

- SQL query construction based on filters
- Dynamic WHERE clauses for flexible filtering
- Cursor-based pagination support
- Optimize with appropriate indexes

**3.3 Cache Layer**

- Redis integration for hot queries
- Cache key design: `query:{hash(params)}:{version}`
- TTL strategy:
  - Hot queries: 30s
  - Stats: 5min
  - Historical: 1hr
- Cache invalidation on new events

**3.4 Query Optimization**

- Use GIN index for JSONB queries (MVP approach)
- Query analyzer for slow query detection
- Connection pooling configuration
- Prepared statements

**3.5 Aggregations**

- `GetContractStats`: totalEvents, latestBlock, indexerDelay
- Efficient COUNT queries with caching
- Support for time-range aggregations

**Key Files**:

- `internal/server/server.go` - gRPC server
- `internal/service/query_service.go` - Core query logic
- `internal/cache/cache.go` - Redis cache manager
- `internal/optimizer/query_builder.go` - SQL builder
- `internal/aggregator/stats.go` - Statistics calculator

**Performance Targets**:

- P50 < 50ms
- P95 < 200ms
- Cache hit rate > 70%

---

### Task 4: API Gateway Implementation

**Location**: `services/api-gateway/`

Build the public-facing API Gateway with GraphQL and REST support.

**Core Components**:

**4.1 GraphQL Resolvers**

- Implement all Query resolvers
- Implement all Mutation resolvers
- Call Query Service via gRPC
- Call Admin Service via gRPC
- Implement DataLoader to prevent N+1 queries

**4.2 gRPC Clients**

- QueryServiceClient connection pool
- AdminServiceClient connection pool
- Retry logic with exponential backoff
- Circuit breaker pattern
- Timeout configuration

**4.3 Middleware**

- Authentication (API Key validation)
- Rate limiting (Redis-based token bucket)
- CORS configuration
- Request logging
- Metrics collection
- Panic recovery

**4.4 REST Endpoints** (Optional but recommended)

```
GET  /api/v1/events
GET  /api/v1/events/tx/:txHash
GET  /api/v1/events/address/:address
GET  /api/v1/contracts
POST /api/v1/contracts
GET  /api/v1/health
```

**4.5 Error Handling**

- Unified error format
- gRPC error translation
- Proper HTTP status codes
- Detailed error messages (dev mode)

**Key Files**:

- `cmd/main.go` - Server entry point
- `internal/resolver/resolver.go` - GraphQL resolvers
- `internal/handler/rest.go` - REST handlers
- `internal/middleware/auth.go` - Authentication
- `internal/middleware/ratelimit.go` - Rate limiting
- `internal/client/grpc_clients.go` - Service clients

**Configuration**:

```yaml
server:
  port: 8000
  cors_origins: ["http://localhost:3000"]
  
grpc:
  query_service: "localhost:8081"
  admin_service: "localhost:8082"
  timeout: 10s
  
rate_limit:
  free_tier: 100  # requests per minute
  pro_tier: 1000
```

---

### Task 5: Admin Service Implementation

**Location**: `services/admin-service/`

Implement management and operational features.

**Core Components**:

**5.1 Contract Management**

- AddContract (idempotent) - check if exists before creating
- RemoveContract - mark as inactive, don't delete
- UpdateContract - modify configuration
- ListContracts - with filters

**5.2 Backfill Manager**

- Task queue using Redis
- Chunk large block ranges (1000 blocks per chunk)
- Worker pool for parallel processing
- Progress tracking and persistence
- Rate limiting to avoid RPC throttling
- Resume capability on restart

**5.3 System Status**

- Indexer lag monitoring
- RPC health status
- Database connection status
- Cache hit rates
- Error log retrieval

**5.4 Alert Manager**

- Alert rules configuration
- Webhook notifications (Slack)
- Alert history

**Key Files**:

- `internal/server/server.go` - gRPC server
- `internal/service/contract_manager.go` - Contract CRUD
- `internal/backfill/manager.go` - Backfill orchestration
- `internal/backfill/worker.go` - Backfill worker
- `internal/monitor/system_status.go` - System monitoring

**Backfill Configuration**:

```yaml
backfill:
  chunk_size: 1000
  max_concurrent_chunks: 3
  rate_limit: 100  # RPC calls per minute
```

---

### Task 6: Integration & Testing

**6.1 Service Integration**

- Start all services with docker-compose
- Verify gRPC connectivity
- Test GraphQL queries through Playground
- Test REST endpoints

**6.2 Unit Tests**

- Query builder tests
- Cache manager tests
- Resolver tests (with mocks)
- Middleware tests

**6.3 Integration Tests**

- End-to-end API tests
- Query Service tests with real database
- Cache invalidation tests
- Backfill workflow tests

**6.4 Performance Tests**

- Load test with k6
- Cache hit rate validation
- Query latency verification
- Concurrent request handling

**Test Coverage Target**: 75%+

---

### Task 7: Documentation & Configuration

**7.1 API Documentation**

- GraphQL schema documentation
- REST API endpoint documentation
- Authentication guide
- Rate limiting explanation
- Example queries and mutations

**7.2 Deployment Configuration**

- Update docker-compose.yml with all services
- Kubernetes manifests (if needed)
- Environment variable templates
- Service health checks

**7.3 Development Guide**

- Setup instructions
- How to add new queries
- How to test locally
- Debugging tips

**Files to Create/Update**:

- `docs/api/graphql-reference.md`
- `docs/api/rest-endpoints.md`
- `docs/api/authentication.md`
- `README.md` (update with API usage)
- `docker-compose.yml` (add API services)

---

## Success Criteria

**Functional Requirements**:

- [ ] GraphQL API accessible at `http://localhost:8000/graphql`
- [ ] All queries return correct data from database
- [ ] Mutations are idempotent
- [ ] Cursor pagination works correctly
- [ ] Cache reduces database load by 50%+
- [ ] Backfill can process 10,000 blocks successfully
- [ ] REST endpoints (if implemented) functional

**Non-Functional Requirements**:

- [ ] API P95 latency < 200ms
- [ ] Cache hit rate > 70% for hot queries
- [ ] All services start without errors
- [ ] gRPC communication works between services
- [ ] Rate limiting prevents abuse
- [ ] Error messages are clear and actionable

**Testing Requirements**:

- [ ] Unit tests pass with 75%+ coverage
- [ ] Integration tests verify end-to-end flows
- [ ] Load test handles 100 concurrent users
- [ ] GraphQL Playground accessible

---

## Key Technical Decisions

### 1. GIN Index vs Dedicated Address Table

**Decision**: Start with GIN index (MVP), add address table in Phase 4 if P95 > 500ms

**Rationale**: Simpler implementation, easier to iterate, sufficient for initial scale

### 2. Cache Strategy

**Decision**: Multi-tier caching (in-memory L1 + Redis L2)

**Rationale**: Balance between speed and consistency

### 3. Pagination

**Decision**: Cursor-based (Relay specification)

**Rationale**: Efficient for large datasets, prevents missing/duplicate records

### 4. Authentication

**Decision**: API Key based (not JWT)

**Rationale**: Simpler for initial version, suitable for service-to-service auth

---

## Dependencies

**External Services**:

- PostgreSQL (from Phase 2) ✅
- Redis for caching
- RPC nodes (Alchemy/Infura)

**Go Libraries**:

- `gqlgen` - GraphQL server
- `gin` - HTTP framework
- `grpc-go` - gRPC
- `go-redis` - Redis client
- `protobuf` - Protocol buffers

---

## Risks & Mitigations

| Risk | Mitigation |

|------|------------|

| N+1 query problem | Implement DataLoader |

| Cache stampede | Use single-flight pattern |

| gRPC timeout | Configure appropriate timeouts + retries |

| Query performance | Start with GIN index, monitor, optimize later |

| Rate limit abuse | Redis-based token bucket |

---

## Next Steps After Phase 3

- **Phase 4**: Performance optimization, monitoring dashboards
- **Phase 5**: Production deployment, CI/CD pipeline
- **Phase 6**: Advanced features (WebSocket subscriptions, multi-chain)

---

## Timeline Estimate

- **Task 1-2** (Schema & Proto): 1-2 days
- **Task 3** (Query Service): 2-3 days
- **Task 4** (API Gateway): 2-3 days
- **Task 5** (Admin Service): 2 days
- **Task 6** (Testing): 2 days
- **Task 7** (Documentation): 1 day

**Total**: 10-13 days (~2 weeks)

### To-dos

- [ ] Design and implement GraphQL schema with all queries, mutations, and types
- [ ] Define gRPC service interfaces for Query and Admin services
- [ ] Implement Query Service gRPC server with core query logic
- [ ] Implement Redis caching layer with invalidation strategy
- [ ] Build query optimizer and SQL builder for efficient queries
- [ ] Implement GraphQL resolvers calling Query and Admin services
- [ ] Implement authentication, rate limiting, and CORS middleware
- [ ] Implement REST API endpoints (optional but recommended)
- [ ] Implement contract management with idempotent operations
- [ ] Implement historical backfill with chunking and progress tracking
- [ ] Write integration tests for all API endpoints and service communication
- [ ] Run load tests and verify performance targets (P95 < 200ms, cache hit > 70%)
- [ ] Write comprehensive API documentation with examples