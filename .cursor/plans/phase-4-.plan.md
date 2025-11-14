Phase 4 targets the Query Service slice of the API layer to hit the system-level latency/caching KPIs set in the PRD (P95 < 200 ms, cache hit >70%) while keeping the architecture laid out in Phase 3 intact (API gateway ↔ Query Service ↔ DB + Redis) (docs/smart_contract_event_indexer_prd.md:27-41, .cursor/plans/phase-3-api-layer-315839c7.plan.md:16-186).
Focus across the sprint should remain on the four workstreams already hinted at in the master plan—gRPC service, query optimizer, cache layer, aggregation—and a cross-cutting validation track (docs/smart_contract_event_indexer_plan.md:440-530, .cursor/plans/phase-3-api-layer-315839c7.plan.md:128-186).
Success gating: each workstream must land measurable deliverables (running server, profiling data, cache dashboards, stats API) plus test hooks that prove compliance with the functional/non-functional criteria defined for the API layer (.cursor/plans/phase-3-api-layer-315839c7.plan.md:392-418).
Workstream 1 – Query Service Core (Days 1-2)

Stand up the dedicated gRPC server with interceptors for logging, metrics, and panic recovery, expose standard health checks, and wire proto-generated handlers to the service implementation (docs/smart_contract_event_indexer_plan.md:442-458, .cursor/plans/phase-3-api-layer-315839c7.plan.md:128-150).
Embed the architectural building blocks—DB pool, Redis client, query builder, aggregator—exactly as modeled so downstream layers plug in without churn (docs/smart_contract_event_indexer_architecture.md:472-521).
Delivery proof: containerized service that registers with buf-generated stubs, responds to health probes, and is callable from the API gateway skeleton via gRPC. Block on this before deeper optimization.
Workstream 2 – Query Optimization Layer (Days 2-3)

Implement the smart query router so trivial filters bypass optimization while heavy address scans go through the optimized paths, starting with GIN-backed JSONB queries and leaving a guardrail to pivot into an event_addresses table if metrics show P95 > 500 ms (docs/smart_contract_event_indexer_plan.md:462-483, .cursor/plans/phase-3-api-layer-315839c7.plan.md:159-170, .cursor/plans/phase-3-api-layer-315839c7.plan.md:424-438).
Add tracing around the SQL builder to log slow plans, capture EXPLAIN output for future tuning, and enforce per-query timeouts so API targets are protected (docs/smart_contract_event_indexer_plan.md:462-475).
Output: documented query builder module with benchmarking artifacts (GIN vs. fallback) and a decision checklist for escalating to the dedicated address table.
Workstream 3 – Cache Layer (Days 3-4)

Apply the multi-tier cache strategy: deterministic key hashing with versioning, TTL tiers (30 s for hot data, 5 min stats, 1 h historical), and pre/post hooks to invalidate on new events (docs/smart_contract_event_indexer_plan.md:487-509, .cursor/plans/phase-3-api-layer-315839c7.plan.md:149-158, docs/smart_contract_event_indexer_architecture.md:552-574).
Implement cache-miss protections (Bloom filter, empty-result caching) and LRU controls per the plan, then expose Redis metrics so the hit-rate KPI can be tracked (docs/smart_contract_event_indexer_plan.md:487-509).
Deliverables: cache manager package, instrumentation dashboards, and automated tests proving eviction + invalidation logic.
Workstream 4 – Aggregations & Contract Stats (Days 4-5)

Build the ContractStats path (total events, latest block, indexer delay, optional unique addresses) plus time-windowed aggregations, aligning to the GraphQL schema expectations (docs/smart_contract_event_indexer_plan.md:513-530, docs/smart_contract_event_indexer_prd.md:404-433).
Reuse the cache layer for stats (longer TTL, explicit busting) and ensure aggregations execute via optimized SQL (materialized views or incremental counters) to avoid blowing the P95 budget (docs/smart_contract_event_indexer_architecture.md:472-521).
Artifacts: aggregation module, schema-compliant gRPC responses, and documentation on extending metrics (Top-N, per-address) for future phases.
Workstream 5 – Validation & Operational Readiness (Days 5-6)

Create unit/integration tests across the service, especially for pagination, cache hit paths, and failure handling, to reach the 75 % coverage expectation (.cursor/plans/phase-3-api-layer-315839c7.plan.md:392-418).
Run targeted load tests (100 concurrent gql queries via gateway hitting Query Service) and capture profiling data; use results to decide if the event address table needs to advance now or can wait (docs/smart_contract_event_indexer_plan.md:462-483, docs/smart_contract_event_indexer_prd.md:38-41).
Document runbooks: deployment checklist, Redis/database dependency configs, and monitoring hooks so Phase 5 (Admin service) can reuse the telemetry.