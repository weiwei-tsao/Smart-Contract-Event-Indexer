# Smart Contract Event Indexer

A blockchain event indexer that monitors smart contracts, stores events in a database, and provides fast query APIs.

## What It Does

```
Blockchain  →  Indexer  →  PostgreSQL  →  GraphQL/REST API  →  Your App
```

**Problem:** Querying blockchain directly is slow and expensive.

**Solution:** This indexer continuously watches the blockchain, stores events in PostgreSQL, and lets you query them instantly via API.

## Quick Start

```bash
# 1. Clone and start
git clone https://github.com/yourusername/smart-contract-event-indexer.git
cd smart-contract-event-indexer
docker-compose up -d

# 2. Run migrations
docker-compose exec migrate /migrate -path /migrations -database "$DATABASE_URL" up

# 3. Check it's working
curl http://localhost:8000/api/v1/health
```

**That's it!** Open http://localhost:8000 to see the dashboard.

## Tech Stack & Why

| Component | Choice | Why |
|-----------|--------|-----|
| **Language** | Go | Fast, concurrent, excellent blockchain libraries |
| **Database** | PostgreSQL | JSONB for flexible event data, GIN indexes for fast queries |
| **Cache** | Redis | Sub-millisecond response for hot queries |
| **API** | GraphQL + REST | GraphQL for flexible queries, REST for simplicity |
| **Blockchain** | go-ethereum | Official Ethereum library, battle-tested |

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    API Gateway (:8000)                  │
│              GraphQL / REST / Rate Limiting             │
└─────────────────────────┬───────────────────────────────┘
                          │ gRPC
          ┌───────────────┼───────────────┐
          ▼               ▼               ▼
   ┌─────────────┐ ┌─────────────┐ ┌─────────────┐
   │   Query     │ │   Admin     │ │   Indexer   │
   │  Service    │ │  Service    │ │  Service    │
   │   (:8081)   │ │   (:8082)   │ │   (:8080)   │
   └──────┬──────┘ └──────┬──────┘ └──────┬──────┘
          │               │               │
          └───────────────┼───────────────┘
                          ▼
              ┌──────────────────────┐
              │  PostgreSQL + Redis  │
              └──────────────────────┘
```

**4 services, each with a single job:**

| Service | Port | Job |
|---------|------|-----|
| Indexer | 8080 | Watch blockchain, parse events, store in DB |
| Query | 8081 | Optimize queries, cache results |
| Admin | 8082 | Manage contracts, run backfills |
| Gateway | 8000 | Public API, auth, rate limiting |

## API Examples

**Get events for a contract:**
```bash
curl "http://localhost:8000/api/v1/events?contract=0x123...&limit=10"
```

**GraphQL query:**
```graphql
query {
  events(filter: { contractAddress: "0x123...", eventName: "Transfer" }) {
    edges {
      node {
        blockNumber
        transactionHash
        args
      }
    }
  }
}
```

## Configuration

Key settings in `.env`:

```bash
# Blockchain connection
RPC_ENDPOINT=http://localhost:8545        # Local: Ganache
# RPC_ENDPOINT=https://eth-mainnet.g.alchemy.com/v2/YOUR_KEY  # Production

# How many blocks to wait before confirming events
INDEXER_DEFAULT_CONFIRM_BLOCKS=6          # 6 blocks = ~72s delay, 99.99% accurate
```

**Confirmation strategies:**

| Mode | Blocks | Delay | Use Case |
|------|--------|-------|----------|
| Realtime | 1 | ~12s | Testing, demos |
| Balanced | 6 | ~72s | Most apps (default) |
| Safe | 12 | ~144s | Financial apps |

## Common Commands

```bash
make dev-up          # Start everything
make dev-down        # Stop everything
make test            # Run tests
make logs            # View logs
```

## Project Structure

```
├── services/
│   ├── indexer-service/    # Blockchain monitoring
│   ├── api-gateway/        # Public API
│   ├── query-service/      # Query optimization
│   └── admin-service/      # Management
├── shared/                 # Shared code (models, config, utils)
├── migrations/             # Database schemas
└── docker-compose.yml      # Local dev environment
```

## Endpoints

| URL | Description |
|-----|-------------|
| http://localhost:8000 | Dashboard |
| http://localhost:8000/playground | GraphQL Playground |
| http://localhost:8000/api/v1/health | Health check |
| http://localhost:8000/api/v1/events | Query events |
| http://localhost:8000/api/v1/contracts | Manage contracts |

## Deployment Cost

Designed to run on free tiers:

| Service | Provider | Cost |
|---------|----------|------|
| App Hosting | Railway | $0-5/mo |
| Database | Supabase | Free (500MB) |
| Cache | Upstash Redis | Free (10K/day) |
| Blockchain RPC | Alchemy | Free (300M CU/mo) |

**Total: $0-5/month**

## Documentation

See [docs/README.md](docs/README.md) for detailed documentation including:
- Database schema
- API reference
- Deployment options
- Design decisions

## License

MIT
