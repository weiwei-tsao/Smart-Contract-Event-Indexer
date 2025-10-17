# Smart Contract Event Indexer - Quick Reference

**Project Type**: 🎯 Portfolio/技能展示项目  
**Target Cost**: **$0-5/月**  
**Status**: ✅ Ready for Development

---

## 💰 Free Tier Stack

| Service | Provider | Limit | Usage |
|---------|----------|-------|-------|
| **RPC** | Alchemy | 300M CU/月 | Blockchain connection |
| **Database** | Supabase | 500MB | Event storage |
| **Cache** | Upstash | 10K cmd/day | Query caching |
| **Hosting** | Railway | $5 credit | App hosting |
| **Monitor** | BetterUptime | Free | Uptime tracking |

**Total**: **$0-5/月** ✅

---

## ⚙️ Configuration Presets

### Confirmation Blocks (Configurable per contract)

| Mode | Blocks | Delay | Accuracy | Use Case |
|------|--------|-------|----------|----------|
| **Realtime** | 1 | ~12s | 99% | Demo, testing |
| **Balanced** (default) | 6 | ~72s | 99.99% | Production |
| **Safe** | 12 | ~144s | 99.9999% | Financial |

**Default**: 6 blocks (balanced)

---

## 📊 Database Schema Key Points

### contracts table
```sql
CREATE TABLE contracts (
    id SERIAL PRIMARY KEY,
    address VARCHAR(42) UNIQUE NOT NULL,
    abi JSONB NOT NULL,
    confirm_blocks INTEGER DEFAULT 6, -- Configurable!
    ...
);
```

### indexer_state table
```sql
CREATE TABLE indexer_state (
    contract_id INTEGER PRIMARY KEY,  -- No separate id!
    last_indexed_block BIGINT NOT NULL,
    ...
);
```

**Key Fix**: `contract_id` is PRIMARY KEY (not separate `id`)

---

## 🚀 Quick Start Commands

```bash
# Development
make dev-up          # Start local environment
make migrate-up      # Run migrations
make test           # Run tests

# Deployment
docker-compose up   # Local test
railway up          # Deploy to Railway
```

---

## 🔑 Key Technical Decisions

### 1. Confirmation Strategy
- **Configurable** (not fixed 12 blocks)
- Default: 6 blocks (~72s delay)
- Set per contract in database

### 2. Cost Optimization
- **Batch RPC calls** (reduce 99% calls)
- **Cache queries** (Redis)
- **Free tiers first**

### 3. Scope
- ✅ MVP: Core indexing + GraphQL API
- ❌ Phase 6+: WebSocket subscriptions
- ❌ Future: Multi-chain support

---

## 📝 Environment Variables

```bash
# RPC
RPC_URL=wss://eth-mainnet.g.alchemy.com/v2/YOUR_KEY
RPC_FALLBACK_1=https://rpc.ankr.com/eth

# Database
DATABASE_URL=postgresql://...  # Supabase connection string
REDIS_URL=redis://...          # Upstash connection string

# Indexer
DEFAULT_CONFIRM_BLOCKS=6
BATCH_SIZE=100
POLL_INTERVAL=6s
```

---

## 🎯 Success Metrics

### Performance Targets
- Indexing delay: ~72s (balanced mode)
- API P95: <200ms
- Uptime: 99%

### Portfolio Goals
- ✅ Demonstrate Web3 skills
- ✅ Show microservices architecture
- ✅ Prove cost optimization
- ✅ Live demo accessible

---

## 🔍 Monitoring

### Key Metrics to Track
- `indexer_lag_seconds` - How far behind chain
- `rpc_calls_total` - RPC usage (watch free tier limit)
- `api_request_duration_seconds` - API performance
- `cache_hit_rate` - Redis effectiveness

### Alerts
- Indexing lag > 5 minutes
- RPC errors > 5%
- API P95 > 500ms

---

## 📚 Documentation

- **PRD**: Requirements and features
- **Plan**: Week-by-week implementation
- **Architecture**: Technical design
- **Review**: Issues and resolutions

---

## 🛠️ Common Tasks

### Add New Contract
```graphql
mutation {
  addContract(
    address: "0x..."
    abi: "[...]"
    startBlock: 12345678
    confirmBlocks: 6  # Optional, defaults to 6
  ) {
    success
    contractId
  }
}
```

### Query Events
```graphql
query {
  events(
    contractAddress: "0x..."
    first: 20
  ) {
    edges {
      node {
        eventName
        blockNumber
        args { key value }
      }
    }
  }
}
```

---

## 🚨 Common Pitfalls

### ❌ Don't
- Use single RPC endpoint (always have fallbacks)
- Index without confirmations (will hit reorgs)
- Forget to batch RPC calls (will exceed free tier)
- Store `big.Int` directly in JSON (loses precision)

### ✅ Do
- Implement RPC fallback logic
- Use configurable confirmation blocks
- Batch events (100-500 per request)
- Convert `big.Int` to string

---

## 🔗 Useful Links

- Alchemy Dashboard: https://dashboard.alchemy.com
- Supabase Dashboard: https://app.supabase.com
- Railway Dashboard: https://railway.app
- Upstash Console: https://console.upstash.com

---

## 🎓 Interview Prep

**When asked about this project, highlight**:

1. **Cost Optimization**: "Runs at $0-5/month using free tiers"
2. **Configurability**: "Flexible confirmation strategy per contract"
3. **Scalability**: "Microservices allow independent scaling"
4. **Best Practices**: "Proper reorg handling, batch processing, caching"

**Technical deep-dive topics**:
- Chain reorganization handling
- gRPC service communication
- GraphQL DataLoader pattern
- Database indexing strategy (GIN vs dedicated tables)
- Free tier optimization techniques

---

**Version**: 1.0  
**Last Updated**: 2025-10-17  
**Status**: ✅ Ready for Development

