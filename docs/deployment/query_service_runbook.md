# Query Service Runbook (Phase 4)

## 1. Pre-flight Checklist
1. Export env vars (examples):
   ```sh
   export QUERY_SERVICE_PORT=8081
   export DATABASE_URL=postgres://user:pass@host:5432/event_indexer?sslmode=disable
   export REDIS_URL=redis://localhost:6379
   export CACHE_TTL=30s
   ```
2. Validate schema migrations applied (`make migrate-up`).
3. Run targeted tests:
   ```sh
   cd services/query-service
   go test ./cmd ./internal/service
   ```

## 2. Start/Stop
```sh
cd services/query-service
EXPORT_LOG_LEVEL=info go run ./cmd --config .env
```
- Health check: `grpcurl localhost:8081 grpc.health.v1.Health/Check`.
- Shutdown: send `SIGINT` or `make stop-query-service` (if defined).

## 3. Smoke Test Commands
- Events query: `grpcurl -d '{"contract_address":"0x...","first":50}' localhost:8081 proto.QueryService/GetEvents`.
- Contract stats: `grpcurl -d '{"contract_address":"0x..."}' localhost:8081 proto.QueryService/GetContractStats`.

## 4. Load Testing Template
1. Seed dataset via `scripts/load_test_fixtures.sh`.
2. Run `scripts/benchmarks/run_query_service_bench.sh --duration 120 --concurrency 100`.
3. Capture metrics: `docker exec postgres psql -c "SELECT * FROM pg_stat_statements ORDER BY total_time DESC LIMIT 5;"`.
4. Store artifacts under `benchmarks/phase4/<date>/` and update docs/development/features/005-phase4-query-service.md.

## 5. Monitoring & Alerts
- Prometheus metrics endpoint: `:8081/metrics` (via sidecar exporter).
- Key metrics: `query_service_request_duration_seconds`, `query_service_cache_hits_total`, `query_service_cache_misses_total`.
- Alert thresholds:
  - P95 latency > 200ms for 5m.
  - Cache hit rate < 50% for hot contracts.
  - Health check failure > 3 consecutive probes.

## 6. Incident Response
1. Check health endpoint & logs.
2. Verify DB/Redis connectivity.
3. Flush cache if stale data suspected: `grpcurl -d '{"contract_address":"0x..."}' localhost:8082 proto.AdminService/FlushContractCache` (placeholder).
4. Scale Query Service via Docker/K8s if CPU > 80% sustained.

## 7. Post-Deployment Validation
- Run smoke queries from section 3.
- Review Grafana dashboard (latency, cache hit, DB connections).
- Update PROGRESS tracker with rollout status.
