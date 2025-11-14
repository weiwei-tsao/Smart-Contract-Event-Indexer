# Query Service Schema Notes (Phase 4)

## 1. Contract Statistics
**GraphQL**
```graphql
query ContractStats($address: Address!) {
  contractStats(address: $address) {
    contractAddress
    totalEvents
    latestBlock
    currentBlock
    indexerDelay
    uniqueAddresses
    lastUpdated
  }
}
```
**gRPC (proto StatsQuery)**
```sh
grpcurl -d '{"contract_address":"0xContract"}' localhost:8081 proto.QueryService/GetContractStats
```
Notes:
- `uniqueAddresses` is optional; omitted when no address-derived stats are available.
- `indexerDelay` represents absolute block lag (`abs(latestBlock - currentBlock)`).
- Responses are cached for `AGGREGATION_CACHE_TTL` (default 5m).

## 2. Aggregation Helpers
### Time Range Buckets
Expose via Admin/CLI for dashboards until GraphQL surfaces them:
```graphql
query Range($addr: Address!, $from: DateTime!, $to: DateTime!) {
  timeRangeStats(contractAddress: $addr, from: $from, to: $to, interval: HOUR) {
    bucketStart
    bucketEnd
    eventCount
  }
}
```
_Backend_: `types.TimeRangeQuery` with interval validation (`minute|hour|day`). TTL inherits from `AGGREGATION_CACHE_TTL`.

### Top Addresses
```graphql
query TopSenders($addr: Address!) {
  topAddresses(contractAddress: $addr, limit: 10, window: "24h") {
    address
    eventCount
  }
}
```
_Backend_: `types.TopNQuery` (window defaults to 24h). Cache key prefix `agg:top`.

## 3. Pagination Notes
- Cursor fields map to internal event IDs; consumers treat them as opaque strings.
- `events` + `eventsByAddress` honor Relay-style params (`first`, `after`, `last`, `before`).
- Server enforces `MAX_QUERY_LIMIT` (1,000 by default) regardless of client input.

## 4. Env/Config Cross-Reference
| Env | Purpose |
|-----|---------|
| `AGGREGATION_CACHE_TTL` | TTL for `contractStats`, time-range, and top-address queries |
| `NEGATIVE_CACHE_TTL` | TTL for empty-result sentinels to avoid hot-miss stampedes |
| `QUERY_TIMEOUT` | gRPC handler deadline; applies to stats + aggregation queries |
| `SLOW_QUERY_THRESHOLD` | Triggers WARN log with EXPLAIN for slow aggregations |

## 5. Future Schema Hooks
- `timeRangeStats` + `topAddresses` resolvers will live under Admin API until GraphQL surfaces them.
- When enabling GraphQL, reuse gRPC payloads (no new DB code required).
- Document updates should include sample requests + caching guidance.
