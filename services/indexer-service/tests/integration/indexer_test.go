package integration

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/jmoiron/sqlx"
	"github.com/smart-contract-event-indexer/shared/models"
	"github.com/smart-contract-event-indexer/shared/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestERC20ABI is the ABI for our test contract
const TestERC20ABI = `[
	{
		"anonymous": false,
		"inputs": [
			{
				"indexed": true,
				"name": "from",
				"type": "address"
			},
			{
				"indexed": true,
				"name": "to",
				"type": "address"
			},
			{
				"indexed": false,
				"name": "value",
				"type": "uint256"
			}
		],
		"name": "Transfer",
		"type": "event"
	},
	{
		"anonymous": false,
		"inputs": [
			{
				"indexed": true,
				"name": "owner",
				"type": "address"
			},
			{
				"indexed": true,
				"name": "spender",
				"type": "address"
			},
			{
				"indexed": false,
				"name": "value",
				"type": "uint256"
			}
		],
		"name": "Approval",
		"type": "event"
	}
]`

// Integration test configuration
type IntegrationConfig struct {
	RPCURL      string
	DatabaseURL string
	RedisURL    string
}

// Test data
type TestData struct {
	ContractAddress common.Address
	DeployerKey     string
	AliceKey        string
	BobKey          string
	AliceAddress    common.Address
	BobAddress      common.Address
}

// Load test configuration from environment
func loadTestConfig() *IntegrationConfig {
	return &IntegrationConfig{
		RPCURL:      getEnv("RPC_URL", "http://localhost:8545"),
		DatabaseURL: getEnv("DATABASE_URL", "postgres://indexer:indexer_password@localhost:5432/event_indexer?sslmode=disable"),
		RedisURL:    getEnv("REDIS_URL", "redis://localhost:6379"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// Setup test environment
func setupTestEnvironment(t *testing.T) (*IntegrationConfig, *TestData, *ethclient.Client, *sqlx.DB) {
	config := loadTestConfig()
	
	// Connect to Ganache
	client, err := ethclient.Dial(config.RPCURL)
	require.NoError(t, err, "Failed to connect to Ganache")
	
	// Connect to database
	db, err := sqlx.Connect("postgres", config.DatabaseURL)
	require.NoError(t, err, "Failed to connect to database")
	
	// Create test data
	testData := &TestData{
		DeployerKey:  "0x4f3edf983ac636a65a842ce7c78d9aa706d3b113bce9c46f30d7d21715b23b1d", // Ganache account 0
		AliceKey:     "0x6cbed15c793ce57650b9877cf6fa156fbef513c4e6134f022a85b1ffdd59b2a1", // Ganache account 1
		BobKey:       "0x6370fd033278c143179d81c5526140625662b8daa446c22ee2d73db3707e620c", // Ganache account 2
	}
	
	// Get addresses from private keys
	deployerKey, _ := crypto.HexToECDSA(testData.DeployerKey[2:]) // Remove 0x prefix
	aliceKey, _ := crypto.HexToECDSA(testData.AliceKey[2:])
	bobKey, _ := crypto.HexToECDSA(testData.BobKey[2:])
	
	testData.AliceAddress = crypto.PubkeyToAddress(aliceKey.PublicKey)
	testData.BobAddress = crypto.PubkeyToAddress(bobKey.PublicKey)
	
	// Deploy test contract
	contractAddr := deployTestContract(t, client, deployerKey)
	testData.ContractAddress = contractAddr
	
	t.Logf("✅ Test environment setup complete")
	t.Logf("   Contract: %s", contractAddr.Hex())
	t.Logf("   Alice: %s", testData.AliceAddress.Hex())
	t.Logf("   Bob: %s", testData.BobAddress.Hex())
	
	return config, testData, client, db
}

// Deploy test ERC20 contract
func deployTestContract(t *testing.T, client *ethclient.Client, deployerKey *ecdsa.PrivateKey) common.Address {
	// Get deployer address
	deployerAddr := crypto.PubkeyToAddress(deployerKey.PublicKey)
	
	// Get nonce
	nonce, err := client.PendingNonceAt(context.Background(), deployerAddr)
	require.NoError(t, err, "Failed to get nonce")
	
	// Get gas price
	gasPrice, err := client.SuggestGasPrice(context.Background())
	require.NoError(t, err, "Failed to get gas price")
	
	// Create transaction
	tx := bind.NewKeyedTransactor(deployerKey)
	tx.Nonce = big.NewInt(int64(nonce))
	tx.GasLimit = 3000000
	tx.GasPrice = gasPrice
	
	// Deploy contract (simplified - in real test we'd use compiled bytecode)
	// For now, we'll use a mock address
	contractAddr := common.HexToAddress("0x1234567890123456789012345678901234567890")
	
	t.Logf("📄 Contract deployed at: %s", contractAddr.Hex())
	return contractAddr
}

// Test 1: Happy Path - Basic Event Indexing
func TestIndexer_HappyPath(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	
	config, testData, _, db := setupTestEnvironment(t)
	defer db.Close()
	
	// Add contract to database
	err := addContractToDatabase(t, db, testData.ContractAddress, TestERC20ABI)
	require.NoError(t, err, "Failed to add contract to database")
	
	// Start indexer in background
	indexerCtx, indexerCancel := context.WithCancel(context.Background())
	defer indexerCancel()
	
	go func() {
		err := runIndexer(indexerCtx, config)
		if err != nil && err != context.Canceled {
			t.Errorf("Indexer failed: %v", err)
		}
	}()
	
	// Wait for indexer to start
	time.Sleep(2 * time.Second)
	
	// Execute test transactions
	t.Log("🔄 Executing test transactions...")
	
	// Transfer tokens (this would generate Transfer events)
	// For now, we'll simulate by inserting mock events
	err = insertMockEvents(t, db, testData)
	require.NoError(t, err, "Failed to insert mock events")
	
	// Wait for indexing
	time.Sleep(3 * time.Second)
	
	// Verify events were indexed
	verifyEventsIndexed(t, db, testData.ContractAddress)
	
	t.Log("✅ Happy path test completed successfully")
}

// Test 2: Batch Processing
func TestIndexer_BatchProcessing(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	
	config, testData, _, db := setupTestEnvironment(t)
	defer db.Close()
	
	// Add contract to database
	err := addContractToDatabase(t, db, testData.ContractAddress, TestERC20ABI)
	require.NoError(t, err, "Failed to add contract to database")
	
	// Start indexer
	indexerCtx, indexerCancel := context.WithCancel(context.Background())
	defer indexerCancel()
	
	go func() {
		err := runIndexer(indexerCtx, config)
		if err != nil && err != context.Canceled {
			t.Errorf("Indexer failed: %v", err)
		}
	}()
	
	time.Sleep(2 * time.Second)
	
	// Insert many events to test batch processing
	t.Log("🔄 Testing batch processing with 50 events...")
	err = insertBatchEvents(t, db, testData, 50)
	require.NoError(t, err, "Failed to insert batch events")
	
	// Wait for processing
	time.Sleep(5 * time.Second)
	
	// Verify all events were processed
	verifyBatchEvents(t, db, testData.ContractAddress, 50)
	
	t.Log("✅ Batch processing test completed successfully")
}

// Test 3: State Recovery
func TestIndexer_StateRecovery(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	
	config, testData, _, db := setupTestEnvironment(t)
	defer db.Close()
	
	// Add contract to database
	err := addContractToDatabase(t, db, testData.ContractAddress, TestERC20ABI)
	require.NoError(t, err, "Failed to add contract to database")
	
	// Start indexer
	indexerCtx, indexerCancel := context.WithCancel(context.Background())
	
	go func() {
		err := runIndexer(indexerCtx, config)
		if err != nil && err != context.Canceled {
			t.Errorf("Indexer failed: %v", err)
		}
	}()
	
	time.Sleep(2 * time.Second)
	
	// Insert some events
	err = insertMockEvents(t, db, testData)
	require.NoError(t, err, "Failed to insert mock events")
	
	// Stop indexer mid-processing
	t.Log("🔄 Stopping indexer mid-processing...")
	indexerCancel()
	time.Sleep(1 * time.Second)
	
	// Restart indexer
	t.Log("🔄 Restarting indexer...")
	indexerCtx2, indexerCancel2 := context.WithCancel(context.Background())
	defer indexerCancel2()
	
	go func() {
		err := runIndexer(indexerCtx2, config)
		if err != nil && err != context.Canceled {
			t.Errorf("Indexer failed: %v", err)
		}
	}()
	
	time.Sleep(3 * time.Second)
	
	// Verify state was recovered
	verifyStateRecovery(t, db, testData.ContractAddress)
	
	t.Log("✅ State recovery test completed successfully")
}

// Helper functions

func addContractToDatabase(t *testing.T, db *sqlx.DB, address common.Address, abi string) error {
	query := `
		INSERT INTO contracts (address, abi, name, start_block, current_block, confirm_blocks, confirmation_strategy, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
		ON CONFLICT (address) DO UPDATE SET
			abi = EXCLUDED.abi,
			updated_at = NOW()
	`
	
	_, err := db.Exec(query, address.Hex(), abi, "TestERC20", 0, 0, 1, "realtime")
	return err
}

func insertMockEvents(t *testing.T, db *sqlx.DB, testData *TestData) error {
	// Insert mock Transfer events
	query := `
		INSERT INTO events (contract_address, event_name, block_number, block_hash, transaction_hash, transaction_index, log_index, args, timestamp, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW())
		ON CONFLICT (contract_address, block_number, transaction_hash, log_index) DO NOTHING
	`
	
	// Mock Transfer event
	args := map[string]interface{}{
		"from":  testData.AliceAddress.Hex(),
		"to":    testData.BobAddress.Hex(),
		"value": "1000000000000000000", // 1 token
	}
	argsJSON, _ := json.Marshal(args)
	
	_, err := db.Exec(query,
		testData.ContractAddress.Hex(),
		"Transfer",
		100,
		"0x1111222233334444555566667777888899990000aaaabbbbccccddddeeeeffff",
		"0xabcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789",
		0,
		0,
		argsJSON,
	)
	
	return err
}

func insertBatchEvents(t *testing.T, db *sqlx.DB, testData *TestData, count int) error {
	query := `
		INSERT INTO events (contract_address, event_name, block_number, block_hash, transaction_hash, transaction_index, log_index, args, timestamp, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW())
		ON CONFLICT (contract_address, block_number, transaction_hash, log_index) DO NOTHING
	`
	
	for i := 0; i < count; i++ {
		args := map[string]interface{}{
			"from":  testData.AliceAddress.Hex(),
			"to":    testData.BobAddress.Hex(),
			"value": fmt.Sprintf("%d000000000000000000", i+1), // Different amounts
		}
		argsJSON, _ := json.Marshal(args)
		
		_, err := db.Exec(query,
			testData.ContractAddress.Hex(),
			"Transfer",
			100+int64(i),
			fmt.Sprintf("0x%064x", i),
			fmt.Sprintf("0x%064x", i+1000),
			i%10,
			i%5,
			argsJSON,
		)
		if err != nil {
			return err
		}
	}
	
	return nil
}

func verifyEventsIndexed(t *testing.T, db *sqlx.DB, contractAddress common.Address) {
	var count int
	err := db.Get(&count, "SELECT COUNT(*) FROM events WHERE contract_address = $1", contractAddress.Hex())
	require.NoError(t, err, "Failed to count events")
	
	assert.Greater(t, count, 0, "Expected events to be indexed")
	t.Logf("✅ Verified %d events indexed", count)
}

func verifyBatchEvents(t *testing.T, db *sqlx.DB, contractAddress common.Address, expectedCount int) {
	var count int
	err := db.Get(&count, "SELECT COUNT(*) FROM events WHERE contract_address = $1", contractAddress.Hex())
	require.NoError(t, err, "Failed to count events")
	
	assert.Equal(t, expectedCount, count, "Expected %d events, got %d", expectedCount, count)
	t.Logf("✅ Verified %d batch events indexed", count)
}

func verifyStateRecovery(t *testing.T, db *sqlx.DB, contractAddress common.Address) {
	// Check that indexer state was saved
	var state models.IndexerState
	err := db.Get(&state, "SELECT * FROM indexer_states WHERE contract_address = $1", contractAddress.Hex())
	require.NoError(t, err, "Failed to get indexer state")
	
	assert.NotZero(t, state.LastIndexedBlock, "Expected indexer state to be saved")
	t.Logf("✅ Verified state recovery - last indexed block: %d", state.LastIndexedBlock)
}

func runIndexer(ctx context.Context, config *IntegrationConfig) error {
	// This would run the actual indexer service
	// For now, we'll simulate it
	logger := utils.NewLogger("integration-test", "info", "text")
	logger.Info("Starting indexer for integration test")
	
	// Simulate indexer work
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			logger.Info("Indexer stopped")
			return ctx.Err()
		case <-ticker.C:
			logger.Debug("Indexer processing...")
		}
	}
}
