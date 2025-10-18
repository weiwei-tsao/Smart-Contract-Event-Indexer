# Architecture Decision Record: Microservices Architecture

**Decision ID**: ADR-001
**Date**: 2025-10-17
**Status**: Accepted
**Deciders**: Project Team

---

## Context

We need to build a Smart Contract Event Indexer that can:
- Monitor blockchain events in real-time
- Parse and store events in a database
- Provide fast querying capabilities via APIs
- Scale independently based on demand
- Handle different types of smart contracts

## Decision

We will use a **microservices architecture** with the following services:
- **Indexer Service**: Monitors blockchain and indexes events
- **API Gateway**: Public-facing API endpoints
- **Query Service**: Optimized data querying with caching
- **Admin Service**: Management and monitoring

## Rationale

### Why Microservices?

1. **Independent Scaling**: Each service can scale based on its specific needs
   - Indexer service needs high CPU for blockchain processing
   - Query service needs high memory for caching
   - API Gateway needs high throughput for requests

2. **Technology Flexibility**: Each service can use the most appropriate technology
   - Go for blockchain processing (performance)
   - Node.js for API Gateway (ecosystem)
   - Rust for query optimization (speed)

3. **Fault Isolation**: Failure in one service doesn't bring down the entire system
   - Indexer can fail without affecting API queries
   - Query service can fail without affecting indexing

4. **Team Scalability**: Different teams can work on different services
   - Blockchain team focuses on indexer
   - API team focuses on gateway and query services

### Why Not Monolithic?

1. **Scaling Challenges**: All components scale together
2. **Technology Lock-in**: Single technology stack for all concerns
3. **Deployment Complexity**: Single point of failure
4. **Team Bottlenecks**: All changes require coordination

### Why Not Serverless?

1. **State Management**: Indexer needs persistent state
2. **Cold Start Issues**: Blockchain monitoring needs consistent uptime
3. **Cost**: Long-running processes are expensive in serverless
4. **Complexity**: Event-driven architecture adds complexity

## Consequences

### Positive
- ✅ Independent scaling and deployment
- ✅ Technology flexibility per service
- ✅ Fault isolation
- ✅ Team autonomy
- ✅ Clear service boundaries

### Negative
- ❌ Increased operational complexity
- ❌ Network latency between services
- ❌ Data consistency challenges
- ❌ More complex debugging
- ❌ Service discovery needed

### Mitigation Strategies
- Use gRPC for internal communication (low latency)
- Implement circuit breakers for fault tolerance
- Use shared database for data consistency
- Implement comprehensive monitoring
- Use service mesh for service discovery

## Implementation

### Service Communication
- **Internal**: gRPC for high-performance communication
- **External**: REST/GraphQL APIs
- **Events**: Redis for pub/sub patterns

### Data Management
- **Shared Database**: PostgreSQL for consistency
- **Caching**: Redis for performance
- **State**: Each service manages its own state

### Deployment
- **Development**: Docker Compose
- **Production**: Kubernetes
- **Monitoring**: Prometheus + Grafana

## Status

This decision was made during Phase 1 and implemented throughout Phase 2. The microservices architecture has proven successful for the indexer service implementation.

## Review

This decision should be reviewed when:
- Adding new services
- Scaling beyond current capacity
- Technology requirements change
- Team structure changes

---

**Next Review Date**: 2025-12-17 (3 months)
