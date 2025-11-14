# Phase 4 – Query Service Enhancements PR

## Summary
Phase 4 focuses on hardening the Query Service so the API layer can hit the PRD latency/caching KPIs. Key delivery areas:
- Query Service core: health service registration, structured config logging, slow-query awareness, cache-bloom protections, and routing between simple/complex query paths.
- Optimization guidance: benchmark plan + decision matrix (docs/development/features/005-phase4-query-service.md) that captures when to promote the `event_addresses` helper table.
- Schema clarity: new `docs/api/query_service_schema.md` outlines how GraphQL/gRPC clients should consume contract stats, time-range buckets, and top-address aggregations.
- Operational readiness: runbook (`docs/deployment/query_service_runbook.md`) plus helper tests for pagination and masking functions.

## Change Highlights
1. **services/query-service**
   - Added cache Bloom filter + negative cache TTL support, new optimizer fast paths, stats aggregations, and health endpoint registration.
   - Structured startup logging in `cmd/main.go` with environment summary + masking helper.
   - Added helper tests for pagination logic and config masking.
2. **Shared docs**
   - Benchmarking/decision doc, schema usage notes, and runbook for deployment/testing.
3. **Tracking**
   - `PHASE_4_TODO.md` updated with remaining actions (load test execution).

## Testing
- `cd services/query-service && go test ./...`
- Manual grpcurl smoke checks (documented in runbook).

## Deployment/Runbook
- See `docs/deployment/query_service_runbook.md` for env variables, smoke tests, monitoring, and incident response guidance.
- Health endpoint: `grpc.health.v1.Health/Check` on port `QUERY_SERVICE_PORT`.

## Risks / Follow-ups
- Need to execute the documented 100-concurrency load test once staging data is refreshed.
- Decision on `event_addresses` table pending next benchmarking run.
