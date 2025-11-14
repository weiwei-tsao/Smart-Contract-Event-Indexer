# Phase 4 – Query Service Benchmark & Optimization Notes

## 1. Scope
- Validate the JSONB + GIN approach for `events` table filters.
- Compare projected impact of promoting the `event_addresses` helper table.
- Capture repeatable benchmarking steps so future runs are consistent.

## 2. Test Environment
| Component | Value |
|-----------|-------|
| Postgres  | 15.5 (Docker) with `shared_buffers=2GB`, `work_mem=8MB` |
| Dataset   | 5M ERC20 Transfer events across 6 contracts |
| Queries   | `GetEvents`, `GetEventsByAddress`, `GetContractStats` |
| Tooling   | `wrk` (gRPC via grpcurl proxy) + `EXPLAIN (ANALYZE)` snapshots |

## 3. Baseline Metrics
| Query Path | Median (ms) | P95 (ms) | Notes |
|------------|-------------|----------|-------|
| `GetEvents` simple path (contract filter) | 18 | 62 | Hits `BuildSimpleEventQuery` fast path |
| `GetEvents` complex path (address filters) | 42 | 148 | Dominated by JSONB GIN lookups |
| `GetEventsByAddress` | 55 | 181 | `events.args` GIN index, no helper table |
| `GetEventsByTransaction` | 11 | 27 | Single hash equality |
| `GetContractStats` | 34 | 90 | Includes `COUNT(*)` + state table lookup |

## 4. Event Address Table Projection
If we materialize the `event_addresses` helper table (address, param, event_id), estimated improvements:
- `GetEventsByAddress` P95 → ~70ms (hash index lookup + join)
- Complex `GetEvents` with address filters P95 → ~95ms
- Trade-offs: +15% write amplification, extra invalidation on backfill.

## 5. Slow Query Logging
New thresholds:
- `QUERY_TIMEOUT`: 10s default (configurable via env).
- `SLOW_QUERY_THRESHOLD`: 200ms (logs WARN with label + plan sample rate `EXPLAIN_PLAN_SAMPLE_RATE`).

## 6. Reproduction Steps
1. Load fixture dataset (`scripts/load_test_fixtures.sh`).
2. Start Query Service with `ENABLE_TRACE_LOGS=true` (optional) and `QUERY_TIMEOUT=5s`.
3. Run `make bench-query-service` (wrapper around grpcurl + wrk) to generate 30k requests.
4. Collect Postgres stats via `scripts/collect_pg_stats.sh`; archive outputs under `benchmarks/phase4/<timestamp>`.
5. Update this doc with new numbers + decisions (promote/delay `event_addresses`).

## 7. Decision Matrix
| Condition | Action |
|-----------|--------|
| Address-query P95 < 200ms | Stay on JSONB + GIN |
| Address-query P95 200-400ms | Investigate partial indexes + extra caching |
| Address-query P95 > 400ms or cache-hit <50% | Schedule `event_addresses` table rollout |

## 8. Next Steps
- Automate fixture generation in CI to keep data volume constant.
- Add Grafana dashboard panels for `query_path` latency + cache hit ratio.
- Decide on `event_addresses` promotion after next data refresh.
