<!-- 419d3919-2e0c-4a54-b7a3-0d87c132dec1 c72834c9-f803-4d71-945c-10fdfeae068d -->
# Phase 2: Indexer Service Core Development

## Stage 2A: Minimal Working Indexer (Priority)

Build a basic but functional indexer that connects to Ganache, parses events, and stores them in PostgreSQL.

### 1. Project Structure Setup

Create the internal package structure in `services/indexer-service/`:

```
internal/
├── blockchain/     # RPC connection & block monitoring
├── parser/         # ABI & event log parsing
├── storage/        # Database operations
└── indexer/        # Main indexer orchestration
cmd/
└── main.go         # Application entry point
```

### 2. Blockchain Connection Module

**File**: `internal/blockchain/client.go`

- Implement `Client` struct with ethclient connection
- Connect to Ganache at `http://localhost:8545`
- Methods: `Connect()`, `GetLatestBlock()`, `SubscribeNewBlocks()`, `GetLogs()`
- Basic error handling and reconnection

**File**: `internal/blockchain/monitor.go`

- Implement `BlockMonitor` that polls for new blocks
- Simple polling loop (check every 6 seconds)
- Return new block numbers for processing

### 3. Event Parsing Module

**File**: `internal/parser/abi.go`

- Load and parse contract ABI JSON
- Extract event definitions from ABI
- Map event signatures to event names
- Use `github.com/ethereum/go-ethereum/accounts/abi` package

**File**: `internal/parser/event.go`

- Implement `EventParser` struct
- Parse `types.Log` into `models.Event`
- Decode indexed and non-indexed parameters
- Handle basic types: address, uint256, string, bytes
- Convert BigInt to string for storage
- Handle address checksum formatting

### 4. Data Persistence Module

**File**: `internal/storage/contract.go`

- `GetContract(address)` - Fetch contract configuration
- `UpdateContractBlock(address, blockNumber)` - Update progress
- Use `shared/database` connection pool

**File**: `internal/storage/event.go`

- `InsertEvents(events []models.Event)` - Batch insert with COPY protocol or multi-row INSERT
- Use `ON CONFLICT DO NOTHING` for idempotency
- Transaction support for atomicity

**File**: `internal/storage/state.go`

- `GetIndexerState(contractAddress)` - Get current indexing progress
- `SaveIndexerState(state)` - Update progress tracking

### 5. Main Indexer Loop

**File**: `internal/indexer/indexer.go`

- Implement `Indexer` struct orchestrating all components
- Main loop:

  1. Get latest block from blockchain
  2. For each monitored contract:

     - Get current indexed block from DB
     - Fetch logs from current+1 to latest block
     - Parse logs into events
     - Batch insert events to DB
     - Update contract current_block

  1. Sleep for poll interval (6s)

- Simple graceful shutdown with context cancellation

**File**: `cmd/main.go`

- Initialize configuration from environment
- Set up database connection
- Set up blockchain client
- Create and start indexer
- Handle OS signals for graceful shutdown

### 6. Configuration

**File**: `internal/config/config.go`

- Load settings from environment variables:
  - `RPC_ENDPOINT` (default: http://localhost:8545)
  - `DATABASE_URL`
  - `POLL_INTERVAL` (default: 6s)
  - `BATCH_SIZE` (default: 100)
- Use `shared/config` utilities

## Stage 2B: Enhancement Features

Add production-ready features for robustness.

### 7. Advanced RPC Management

**File**: `internal/blockchain/manager.go`

- Implement `RPCManager` with fallback nodes support
- Health check mechanism for RPC endpoints
- Automatic failover on connection errors
- Support for WebSocket subscriptions (prepare for future)
- Configuration for primary + fallback RPC endpoints

### 8. Reorg Detection & Handling

**File**: `internal/reorg/detector.go`

- Cache last 50 block hashes in Redis (use `shared/database/redis.go`)
- On each new block, verify parent hash matches cache
- Detect fork point when mismatch occurs

**File**: `internal/reorg/handler.go`

- Rollback database: delete events where block_number > fork_point
- Reset contract current_block to fork_point
- Trigger reindexing from fork_point
- Log reorg events for monitoring

### 9. Confirmation Strategy

**File**: `internal/indexer/confirmation.go`

- Implement confirmation checking logic
- Only process events from blocks with sufficient confirmations
- Calculate: `latestBlock - eventBlock >= contract.ConfirmBlocks`
- Support different strategies per contract (1/6/12 blocks)

### 10. Graceful Shutdown & Recovery

**File**: `internal/indexer/lifecycle.go`

- Implement proper context propagation
- Wait for current batch to complete before shutdown
- Save state before exit
- Resume from last saved state on restart
- Handle partial batch scenarios

### 11. Error Handling & Retry

**File**: `internal/indexer/retry.go`

- Exponential backoff for transient errors
- Classify errors: retriable vs fatal
- Max retry attempts configuration
- Circuit breaker for repeated failures

## Stage 2C: Testing

Comprehensive testing after implementation is complete.

### 12. Unit Tests

Create test files for each module:

- `internal/parser/event_test.go` - Test event parsing with mock logs
- `internal/blockchain/client_test.go` - Test with mock RPC responses
- `internal/storage/*_test.go` - Test with test database
- Use table-driven tests for multiple scenarios
- Mock external dependencies
- Target: 75%+ coverage on business logic

### 13. Integration Tests

**File**: `tests/integration/indexer_test.go`

- Use Testcontainers for PostgreSQL
- Deploy test ERC20 contract to Ganache
- Trigger Transfer events
- Verify events are indexed correctly
- Test end-to-end flow

### 14. End-to-End Demo

Create a demo script that:

1. Deploys a sample ERC20 contract to Ganache
2. Starts the indexer service
3. Executes token transfers
4. Queries the database to verify indexed events
5. Demonstrates reorg handling (if time permits)

**File**: `tests/e2e/demo.sh` - Automation script

**File**: `tests/e2e/contracts/SampleERC20.sol` - Test contract

## Key Implementation Notes

- Use `github.com/ethereum/go-ethereum` for all Ethereum interactions
- Leverage existing `shared/models`, `shared/database`, `shared/utils`
- All timestamps should be UTC
- BigInt values stored as strings in JSONB
- Use structured logging with context
- Configuration via environment variables with sensible defaults
- Database transactions for batch operations

## Success Criteria

**Stage 2A Complete**:

- Indexer connects to Ganache successfully
- Can parse ERC20 Transfer events
- Events stored in PostgreSQL with correct data
- Service can be started with `make run-indexer`

**Stage 2B Complete**:

- Reorg detection and handling works
- Confirmation strategies implemented
- Graceful shutdown preserves state
- Service recovers from crashes

**Stage 2C Complete**:

- Unit tests pass with 75%+ coverage
- Integration tests verify end-to-end functionality
- Demo successfully shows event indexing

## File Creation Order

1. Configuration and setup (cmd/main.go, internal/config/)
2. Blockchain client (internal/blockchain/client.go)
3. Event parser (internal/parser/)
4. Storage layer (internal/storage/)
5. Main indexer (internal/indexer/indexer.go)
6. Enhancement features (internal/reorg/, blockchain/manager.go)
7. Testing (unit → integration → e2e)

### To-dos

- [ ] Remember to check project rules before coding, such as git conventions, documentations
- [ ] Create internal package structure (blockchain, parser, storage, indexer) and cmd directory
- [ ] Implement configuration loading from environment variables
- [ ] Implement basic blockchain client for Ganache connection
- [ ] Implement block monitoring with polling mechanism
- [ ] Implement ABI parsing to extract event definitions
- [ ] Implement event log parser to decode events from blockchain logs
- [ ] Implement contract storage operations (get, update progress)
- [ ] Implement event batch insertion with idempotency
- [ ] Implement indexer state persistence
- [ ] Implement main indexer orchestration loop
- [ ] Create application entry point with initialization and signal handling
- [ ] Implement advanced RPC manager with fallback support
- [ ] Implement reorg detection using block hash cache
- [ ] Implement reorg handling with database rollback
- [ ] Implement confirmation strategy checking
- [ ] Implement graceful shutdown and state recovery
- [ ] Implement error classification and retry logic
- [ ] Write unit tests for all modules
- [ ] Write integration tests with real dependencies
- [ ] Create end-to-end demo with sample contract