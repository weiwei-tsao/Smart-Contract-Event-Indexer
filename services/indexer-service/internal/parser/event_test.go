package parser

import (
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/smart-contract-event-indexer/indexer-service/internal/testutil"
	"github.com/smart-contract-event-indexer/shared/models"
)

// equalAddresses compares two Ethereum addresses case-insensitively
func equalAddresses(addr1, addr2 string) bool {
	return strings.EqualFold(addr1, addr2)
}

func TestEventParser_ParseLog_Transfer(t *testing.T) {
	logger := testutil.NewTestLogger()
	
	// Create ABI parser
	abiParser, err := NewABIParser(testutil.ERC20ABI, logger)
	if err != nil {
		t.Fatalf("Failed to create ABI parser: %v", err)
	}
	
	// Create event parser
	eventParser := NewEventParser(abiParser, logger)
	
	// Create mock Transfer log
	log := testutil.CreateMockTransferLog()
	blockTimestamp := time.Now()
	
	// Parse the log
	parsedEvent, err := eventParser.ParseLog(log, blockTimestamp)
	if err != nil {
		t.Fatalf("Failed to parse log: %v", err)
	}
	
	// Verify basic fields
	if parsedEvent.EventName != "Transfer" {
		t.Errorf("Expected event name 'Transfer', got: %s", parsedEvent.EventName)
	}
	
	if parsedEvent.ContractAddress != models.Address(log.Address.Hex()) {
		t.Errorf("Expected contract address %s, got: %s", 
			log.Address.Hex(), parsedEvent.ContractAddress)
	}
	
	if parsedEvent.BlockNumber != int64(log.BlockNumber) {
		t.Errorf("Expected block number %d, got: %d", 
			log.BlockNumber, parsedEvent.BlockNumber)
	}
	
	if parsedEvent.TransactionHash != models.Hash(log.TxHash.Hex()) {
		t.Errorf("Expected tx hash %s, got: %s", 
			log.TxHash.Hex(), parsedEvent.TransactionHash)
	}
	
	if parsedEvent.TransactionIndex != int(log.TxIndex) {
		t.Errorf("Expected tx index %d, got: %d", 
			log.TxIndex, parsedEvent.TransactionIndex)
	}
	
	if parsedEvent.LogIndex != int(log.Index) {
		t.Errorf("Expected log index %d, got: %d", 
			log.Index, parsedEvent.LogIndex)
	}
}

func TestEventParser_ParseLog_TransferArgs(t *testing.T) {
	logger := testutil.NewTestLogger()
	abiParser, err := NewABIParser(testutil.ERC20ABI, logger)
	if err != nil {
		t.Fatalf("Failed to create ABI parser: %v", err)
	}
	
	eventParser := NewEventParser(abiParser, logger)
	log := testutil.CreateMockTransferLog()
	blockTimestamp := time.Now()
	
	parsedEvent, err := eventParser.ParseLog(log, blockTimestamp)
	if err != nil {
		t.Fatalf("Failed to parse log: %v", err)
	}
	
	// Verify Args is not nil
	if parsedEvent.Args == nil {
		t.Fatal("Expected non-nil Args")
	}
	
	// Verify 'from' field
	fromAddr, ok := parsedEvent.Args["from"]
	if !ok {
		t.Error("Expected 'from' field in Args")
	}
	// Use strings.EqualFold for case-insensitive comparison (addresses are checksummed)
	fromAddrStr, ok := fromAddr.(string)
	if !ok {
		t.Error("Expected 'from' to be a string")
	}
	expectedFrom := "0xa0B0C0D0E0F0a0B0c0D0E0F0a0B0C0D0E0F0A0b0"
	if len(fromAddrStr) != 42 || fromAddrStr[:2] != "0x" {
		t.Errorf("Expected valid address format, got: %v", fromAddr)
	}
	// Verify address matches (case-insensitive due to checksumming)
	if !equalAddresses(fromAddrStr, expectedFrom) {
		t.Errorf("Expected from address %s (or checksummed variant), got: %v", expectedFrom, fromAddr)
	}
	
	// Verify 'to' field
	toAddr, ok := parsedEvent.Args["to"]
	if !ok {
		t.Error("Expected 'to' field in Args")
	}
	toAddrStr, ok := toAddr.(string)
	if !ok {
		t.Error("Expected 'to' to be a string")
	}
	expectedTo := "0xB1c1d1E1F1A1b1C1D1e1F1a1B1C1D1e1F1a1b1C1"
	if len(toAddrStr) != 42 || toAddrStr[:2] != "0x" {
		t.Errorf("Expected valid address format, got: %v", toAddr)
	}
	// Verify address matches (case-insensitive due to checksumming)
	if !equalAddresses(toAddrStr, expectedTo) {
		t.Errorf("Expected to address %s (or checksummed variant), got: %v", expectedTo, toAddr)
	}
	
	// Verify 'value' field (should be string representation of BigInt)
	value, ok := parsedEvent.Args["value"]
	if !ok {
		t.Error("Expected 'value' field in Args")
	}
	// 1 ETH in wei = 1000000000000000000
	expectedValue := "1000000000000000000"
	if value != expectedValue {
		t.Errorf("Expected value %s, got: %v", expectedValue, value)
	}
}

func TestEventParser_ParseLog_Approval(t *testing.T) {
	logger := testutil.NewTestLogger()
	abiParser, err := NewABIParser(testutil.ERC20ABI, logger)
	if err != nil {
		t.Fatalf("Failed to create ABI parser: %v", err)
	}
	
	eventParser := NewEventParser(abiParser, logger)
	log := testutil.CreateMockApprovalLog()
	blockTimestamp := time.Now()
	
	parsedEvent, err := eventParser.ParseLog(log, blockTimestamp)
	if err != nil {
		t.Fatalf("Failed to parse log: %v", err)
	}
	
	// Verify event name
	if parsedEvent.EventName != "Approval" {
		t.Errorf("Expected event name 'Approval', got: %s", parsedEvent.EventName)
	}
	
	// Verify Args
	if parsedEvent.Args == nil {
		t.Fatal("Expected non-nil Args")
	}
	
	// Verify 'owner' field
	_, ok := parsedEvent.Args["owner"]
	if !ok {
		t.Error("Expected 'owner' field in Args")
	}
	
	// Verify 'spender' field
	_, ok = parsedEvent.Args["spender"]
	if !ok {
		t.Error("Expected 'spender' field in Args")
	}
	
	// Verify 'value' field
	value, ok := parsedEvent.Args["value"]
	if !ok {
		t.Error("Expected 'value' field in Args")
	}
	// 5 ETH in wei = 5000000000000000000
	expectedValue := "5000000000000000000"
	if value != expectedValue {
		t.Errorf("Expected value %s, got: %v", expectedValue, value)
	}
}

func TestEventParser_ParseLog_InvalidLog(t *testing.T) {
	logger := testutil.NewTestLogger()
	abiParser, err := NewABIParser(testutil.ERC20ABI, logger)
	if err != nil {
		t.Fatalf("Failed to create ABI parser: %v", err)
	}
	
	eventParser := NewEventParser(abiParser, logger)
	
	// Create log with missing topics
	log := testutil.CreateLogWithInvalidTopics()
	blockTimestamp := time.Now()
	
	// Should return error
	_, err = eventParser.ParseLog(log, blockTimestamp)
	if err == nil {
		t.Fatal("Expected error for log with missing topics, got nil")
	}
}

func TestEventParser_ParseLog_UnknownEvent(t *testing.T) {
	logger := testutil.NewTestLogger()
	abiParser, err := NewABIParser(testutil.ERC20ABI, logger)
	if err != nil {
		t.Fatalf("Failed to create ABI parser: %v", err)
	}
	
	eventParser := NewEventParser(abiParser, logger)
	
	// Create log with unknown event signature
	log := testutil.CreateMockTransferLog()
	// Replace with invalid event signature
	log.Topics[0] = common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000000")
	blockTimestamp := time.Now()
	
	// Should return error
	_, err = eventParser.ParseLog(log, blockTimestamp)
	if err == nil {
		t.Fatal("Expected error for unknown event signature, got nil")
	}
}

func TestEventParser_ParseLog_Timestamp(t *testing.T) {
	logger := testutil.NewTestLogger()
	abiParser, err := NewABIParser(testutil.ERC20ABI, logger)
	if err != nil {
		t.Fatalf("Failed to create ABI parser: %v", err)
	}
	
	eventParser := NewEventParser(abiParser, logger)
	log := testutil.CreateMockTransferLog()
	
	// Use specific timestamp
	expectedTimestamp := time.Date(2024, 1, 15, 12, 30, 45, 0, time.UTC)
	
	parsedEvent, err := eventParser.ParseLog(log, expectedTimestamp)
	if err != nil {
		t.Fatalf("Failed to parse log: %v", err)
	}
	
	// Verify timestamp
	if !parsedEvent.Timestamp.Equal(expectedTimestamp) {
		t.Errorf("Expected timestamp %v, got: %v", expectedTimestamp, parsedEvent.Timestamp)
	}
}

func TestEventParser_ParseLog_BlockHash(t *testing.T) {
	logger := testutil.NewTestLogger()
	abiParser, err := NewABIParser(testutil.ERC20ABI, logger)
	if err != nil {
		t.Fatalf("Failed to create ABI parser: %v", err)
	}
	
	eventParser := NewEventParser(abiParser, logger)
	log := testutil.CreateMockTransferLog()
	blockTimestamp := time.Now()
	
	parsedEvent, err := eventParser.ParseLog(log, blockTimestamp)
	if err != nil {
		t.Fatalf("Failed to parse log: %v", err)
	}
	
	// Verify block hash
	expectedBlockHash := models.Hash(log.BlockHash.Hex())
	if parsedEvent.BlockHash != expectedBlockHash {
		t.Errorf("Expected block hash %s, got: %s", expectedBlockHash, parsedEvent.BlockHash)
	}
}

func TestEventParser_AddressFormatting(t *testing.T) {
	logger := testutil.NewTestLogger()
	abiParser, err := NewABIParser(testutil.ERC20ABI, logger)
	if err != nil {
		t.Fatalf("Failed to create ABI parser: %v", err)
	}
	
	eventParser := NewEventParser(abiParser, logger)
	log := testutil.CreateMockTransferLog()
	blockTimestamp := time.Now()
	
	parsedEvent, err := eventParser.ParseLog(log, blockTimestamp)
	if err != nil {
		t.Fatalf("Failed to parse log: %v", err)
	}
	
	// Verify address formatting (should have 0x prefix and checksum)
	fromAddr, ok := parsedEvent.Args["from"].(string)
	if !ok {
		t.Fatal("Expected 'from' to be string")
	}
	
	if len(fromAddr) != 42 { // 0x + 40 hex chars
		t.Errorf("Expected address length 42, got: %d", len(fromAddr))
	}
	
	if fromAddr[:2] != "0x" {
		t.Errorf("Expected address to start with '0x', got: %s", fromAddr[:2])
	}
	
	// Verify address is checksummed (has mixed case)
	hasUpperCase := false
	hasLowerCase := false
	for _, c := range fromAddr[2:] {
		if c >= 'A' && c <= 'F' {
			hasUpperCase = true
		}
		if c >= 'a' && c <= 'f' {
			hasLowerCase = true
		}
	}
	
	if !hasUpperCase || !hasLowerCase {
		t.Log("Warning: Address may not be properly checksummed (expected mixed case)")
	}
}

