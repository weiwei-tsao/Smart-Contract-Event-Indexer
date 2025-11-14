# Phase 4 TODO – Query Service Delivery

## Workstream 1 – Query Service Core
- [x] Verify gRPC interceptors (logging, metrics, recovery) align with phase-3 architecture refs and keep configs centralized (docs/smart_contract_event_indexer_plan.md:442-458).
- [x] Expose an official gRPC health endpoint backed by DB/Redis checks so API gateway + ops can probe readiness (docs/smart_contract_event_indexer_plan.md:442-458).
- [x] Wire server bootstrap (cmd/main.go) to surface structured startup/shutdown logs and document env/config expectations.

## Workstream 2 – Query Optimization Layer
- [x] Harden `internal/optimizer` with smart routing paths (simple vs complex filters) and capture EXPLAIN plans for profiling (docs/smart_contract_event_indexer_plan.md:462-475).
- [x] Implement slow-query logging + timeout controls tied to PRD latency targets (docs/smart_contract_event_indexer_prd.md:27-41).
- [ ] Produce benchmarking notes comparing baseline JSONB GIN queries against the planned `event_addresses` table fallback.

## Workstream 3 – Cache Layer Implementation
- [x] Finalize deterministic cache key format + versioning for events/address/stats queries (docs/smart_contract_event_indexer_plan.md:487-509).
- [x] Add Bloom-filter/empty-result protections to curb cache penetration and wire Redis metrics exports.
- [x] Codify invalidation hooks so new events/backfills bust relevant keys (docs/smart_contract_event_indexer_architecture.md:552-574).

## Workstream 4 – Aggregations & Contract Stats
- [x] Flesh out `GetContractStats` with accurate `currentBlock`, `indexerDelay`, and optional unique-address counters (docs/smart_contract_event_indexer_plan.md:513-530).
- [x] Add reusable aggregation helpers for time-range + top-N queries, backed by cache tier with longer TTLs.
- [ ] Document aggregation schema/usage for GraphQL + gRPC clients (docs/smart_contract_event_indexer_prd.md:404-433).

## Workstream 5 – Validation & Operational Readiness
- [ ] Author targeted unit/integration tests for pagination, cache hit/miss paths, and stats to reach 75%+ coverage (.cursor/plans/phase-3-api-layer-315839c7.plan.md:392-418).
- [ ] Run load tests (≥100 concurrent requests) capturing latency + cache hit metrics; attach findings to docs.
- [ ] Publish ops runbooks covering deployment checklist, Redis/DB dependencies, and monitoring hooks for Phase 5 handoff.
