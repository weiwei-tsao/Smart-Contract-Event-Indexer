# Indexer Service - Testing Strategy

**Target**: 75%+ code coverage  
**Approach**: Unit tests first, then integration tests

---

## Unit Test Coverage Plan

### 1. Parser Module (`internal/parser/`)

#### `abi_test.go`
- ✅ Test ABI JSON parsing (valid/invalid)
- ✅ Test event extraction from ABI
- ✅ Test event ID calculation (topic0)
- ✅ Test event lookup by ID and name

#### `event_test.go`
- ✅ Test ERC20 Transfer event parsing
- ✅ Test indexed vs non-indexed parameters
- ✅ Test BigInt conversion to string
- ✅ Test address formatting
- ✅ Test JSONB args structure
- ✅ Edge cases: empty logs, invalid data

### 2. Indexer Module (`internal/indexer/`)

#### `confirmation_test.go`
- ✅ Test confirmation calculation (realtime/balanced/safe)
- ✅ Test block confirmation checking
- ✅ Test edge cases (same block, far future)

#### `retry_test.go`
- ✅ Test error classification (retriable vs fatal)
- ✅ Test exponential backoff calculation
- ✅ Test circuit breaker state transitions
- ✅ Test retry execution with mock functions

### 3. Blockchain Module (`internal/blockchain/`)

#### `client_test.go` (with mocks)
- Mock ethclient for testing
- Test connection initialization
- Test block number retrieval
- Test error handling

#### `monitor_test.go` (with mocks)
- Test polling mechanism
- Test block detection
- Test context cancellation

### 4. Storage Module (`internal/storage/`)

**Note**: These require database mocks or test database

#### `contract_test.go`
- Test GetContract with sqlmock
- Test UpdateContractCurrentBlock
- Test error handling

#### `event_test.go`
- Test BatchInsertEvents with sqlmock
- Test INSERT conflict handling
- Test transaction rollback

#### `state_test.go`
- Test GetIndexerState
- Test UpdateIndexerState
- Test state transitions

### 5. Reorg Module (`internal/reorg/`)

#### `detector_test.go` (with Redis mock)
- Test block hash caching
- Test reorg detection logic
- Test fork point identification

#### `handler_test.go` (with DB mock)
- Test database rollback
- Test state reset
- Test reindexing trigger

---

## Testing Approach

### Dependencies & Mocking:

**No external dependencies** (can test directly):
- ✅ Parser module (uses go-ethereum ABI decoder only)
- ✅ Confirmation logic
- ✅ Retry logic & circuit breaker

**Need mocking**:
- Blockchain client (mock ethclient.Client)
- Storage (use sqlmock or testify/mock)
- Redis (use miniredis or mock)

### Test Utilities:

Create `internal/testutil/` package with:
- Mock logger
- Test fixtures (sample ABIs, events)
- Helper functions for assertions

---

## Integration Test Plan

**File**: `tests/integration/indexer_test.go`

### Setup:
1. Use Testcontainers for PostgreSQL
2. Connect to running Ganache instance
3. Run migrations

### Test Scenarios:
1. **Happy Path**:
   - Deploy ERC20 contract
   - Execute transfers
   - Verify events indexed correctly

2. **Batch Processing**:
   - Generate 100+ events
   - Verify batch insertion works

3. **State Recovery**:
   - Stop indexer mid-batch
   - Restart and verify resume

4. **Reorg Handling** (if time):
   - Simulate chain reorg
   - Verify rollback and reindex

---

## Test Execution Order

### Phase 1: Unit Tests (No Dependencies)
1. ✅ `parser/abi_test.go`
2. ✅ `parser/event_test.go`
3. ✅ `indexer/confirmation_test.go`
4. ✅ `indexer/retry_test.go`

### Phase 2: Unit Tests (With Mocks)
5. ⏳ `blockchain/client_test.go`
6. ⏳ `storage/contract_test.go`
7. ⏳ `storage/event_test.go`
8. ⏳ `reorg/detector_test.go`

### Phase 3: Integration Tests
9. ⏳ `tests/integration/indexer_test.go`

---

## Success Criteria

- ✅ All tests pass
- ✅ 75%+ code coverage
- ✅ No race conditions (run with `-race` flag)
- ✅ Tests complete in <30 seconds
- ✅ Integration tests verify end-to-end flow

---

## Commands

```bash
# Run all unit tests
cd services/indexer-service
go test ./internal/... -v

# Run with coverage
go test ./internal/... -cover -coverprofile=coverage.out

# View coverage report
go tool cover -html=coverage.out

# Run integration tests
go test ./tests/integration/... -v -tags=integration

# Run all tests
go test ./... -v
```

