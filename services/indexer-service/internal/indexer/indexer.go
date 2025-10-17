package indexer

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/smart-contract-event-indexer/indexer-service/internal/blockchain"
	"github.com/smart-contract-event-indexer/indexer-service/internal/parser"
	"github.com/smart-contract-event-indexer/indexer-service/internal/storage"
	"github.com/smart-contract-event-indexer/shared/models"
	"github.com/smart-contract-event-indexer/shared/utils"
)

// Indexer is the main orchestrator for blockchain event indexing
type Indexer struct {
	client          *blockchain.Client
	contractStorage *storage.ContractStorage
	eventStorage    *storage.EventStorage
	stateStorage    *storage.StateStorage
	pollInterval    time.Duration
	batchSize       int
	logger          *utils.Logger
	
	// Contract-specific parsers
	parsersMu sync.RWMutex
	parsers   map[models.Address]*parser.EventParser
}

// NewIndexer creates a new indexer
func NewIndexer(
	client *blockchain.Client,
	contractStorage *storage.ContractStorage,
	eventStorage *storage.EventStorage,
	stateStorage *storage.StateStorage,
	pollInterval time.Duration,
	batchSize int,
	logger utils.Logger,
) *Indexer {
	return &Indexer{
		client:          client,
		contractStorage: contractStorage,
		eventStorage:    eventStorage,
		stateStorage:    stateStorage,
		pollInterval:    pollInterval,
		batchSize:       batchSize,
		logger:          logger,
		parsers:         make(map[models.Address]*parser.EventParser),
	}
}

// Start begins the indexing process
func (i *Indexer) Start(ctx context.Context) error {
	i.logger.Info("Starting indexer")
	
	// Load all contracts to monitor
	contracts, err := i.contractStorage.GetAllContracts(ctx)
	if err != nil {
		return fmt.Errorf("failed to load contracts: %w", err)
	}
	
	if len(contracts) == 0 {
		i.logger.Warn("No contracts to monitor. Add contracts via the admin API.")
	} else {
		i.logger.WithField("contract_count", len(contracts)).Info("Loaded contracts to monitor")
	}
	
	// Initialize parsers for all contracts
	if err := i.initializeParsers(contracts); err != nil {
		return fmt.Errorf("failed to initialize parsers: %w", err)
	}
	
	// Start the main indexing loop
	ticker := time.NewTicker(i.pollInterval)
	defer ticker.Stop()
	
	i.logger.WithField("poll_interval", i.pollInterval).Info("Indexer main loop started")
	
	for {
		select {
		case <-ctx.Done():
			i.logger.Info("Indexer stopping")
			return ctx.Err()
			
		case <-ticker.C:
			if err := i.processAllContracts(ctx); err != nil {
				i.logger.WithError(err).Error("Error processing contracts")
				// Continue despite errors
			}
		}
	}
}

// initializeParsers creates event parsers for all contracts
func (i *Indexer) initializeParsers(contracts []*models.Contract) error {
	i.parsersMu.Lock()
	defer i.parsersMu.Unlock()
	
	for _, contract := range contracts {
		if err := i.createParserForContract(contract); err != nil {
			i.logger.WithError(err).WithField("contract", contract.Address).Error("Failed to create parser")
			continue
		}
	}
	
	return nil
}

// createParserForContract creates an event parser for a specific contract
func (i *Indexer) createParserForContract(contract *models.Contract) error {
	abiParser, err := parser.NewABIParser(contract.ABI, i.logger)
	if err != nil {
		return fmt.Errorf("failed to create ABI parser: %w", err)
	}
	
	eventParser := parser.NewEventParser(abiParser, i.logger)
	i.parsers[contract.Address] = eventParser
	
	i.logger.WithFields(map[string]interface{}{
		"contract": contract.Address,
		"name":     contract.Name,
	}).Debug("Parser created for contract")
	
	return nil
}

// getParserForContract retrieves the parser for a contract
func (i *Indexer) getParserForContract(address models.Address) *parser.EventParser {
	i.parsersMu.RLock()
	defer i.parsersMu.RUnlock()
	return i.parsers[address]
}

// processAllContracts processes all monitored contracts
func (i *Indexer) processAllContracts(ctx context.Context) error {
	// Get latest block from blockchain
	latestBlock, err := i.client.GetLatestBlockNumber(ctx)
	if err != nil {
		return fmt.Errorf("failed to get latest block: %w", err)
	}
	
	// Get all contracts
	contracts, err := i.contractStorage.GetAllContracts(ctx)
	if err != nil {
		return fmt.Errorf("failed to get contracts: %w", err)
	}
	
	// Process each contract
	for _, contract := range contracts {
		if err := i.processContract(ctx, contract, latestBlock); err != nil {
			i.logger.WithError(err).WithFields(map[string]interface{}{
				"contract": contract.Address,
				"name":     contract.Name,
			}).Error("Failed to process contract")
			
			// Record error but continue with other contracts
			i.stateStorage.IncrementErrorCount(ctx, contract.Address, err.Error())
			continue
		}
	}
	
	return nil
}

// processContract processes a single contract
func (i *Indexer) processContract(ctx context.Context, contract *models.Contract, latestBlock int64) error {
	// Get the parser for this contract
	eventParser := i.getParserForContract(contract.Address)
	if eventParser == nil {
		// Parser not found, try to create it
		if err := i.createParserForContract(contract); err != nil {
			return fmt.Errorf("failed to create parser: %w", err)
		}
		eventParser = i.getParserForContract(contract.Address)
	}
	
	// Calculate the block range to index
	fromBlock := contract.CurrentBlock + 1
	
	// Apply confirmation blocks (don't index blocks that aren't confirmed yet)
	confirmedBlock := latestBlock - int64(contract.ConfirmBlocks)
	if confirmedBlock < fromBlock {
		// No new confirmed blocks to process
		return nil
	}
	
	// Limit the batch size
	toBlock := fromBlock + int64(i.batchSize) - 1
	if toBlock > confirmedBlock {
		toBlock = confirmedBlock
	}
	
	// Skip if no blocks to process
	if fromBlock > toBlock {
		return nil
	}
	
	i.logger.WithFields(map[string]interface{}{
		"contract":   contract.Address,
		"from_block": fromBlock,
		"to_block":   toBlock,
		"latest":     latestBlock,
		"confirmed":  confirmedBlock,
	}).Debug("Processing contract")
	
	// Fetch logs from blockchain
	logs, err := i.client.GetLogsForContract(
		ctx,
		common.HexToAddress(string(contract.Address)),
		fromBlock,
		toBlock,
	)
	if err != nil {
		return fmt.Errorf("failed to get logs: %w", err)
	}
	
	if len(logs) == 0 {
		i.logger.WithFields(map[string]interface{}{
			"contract":   contract.Address,
			"from_block": fromBlock,
			"to_block":   toBlock,
		}).Debug("No logs found in block range")
		
		// Update current block even if no logs
		if err := i.contractStorage.UpdateContractBlock(ctx, contract.Address, toBlock); err != nil {
			return fmt.Errorf("failed to update contract block: %w", err)
		}
		
		return nil
	}
	
	// Get block timestamp for the last block in the range
	block, err := i.client.GetBlockByNumber(ctx, toBlock)
	if err != nil {
		return fmt.Errorf("failed to get block: %w", err)
	}
	blockTimestamp := time.Unix(int64(block.Time()), 0).UTC()
	
	// Parse logs into events
	events, err := eventParser.ParseLogs(logs, blockTimestamp)
	if err != nil {
		return fmt.Errorf("failed to parse logs: %w", err)
	}
	
	if len(events) == 0 {
		i.logger.WithField("contract", contract.Address).Debug("No events parsed from logs")
		
		// Update current block
		if err := i.contractStorage.UpdateContractBlock(ctx, contract.Address, toBlock); err != nil {
			return fmt.Errorf("failed to update contract block: %w", err)
		}
		
		return nil
	}
	
	// Insert events into database
	if err := i.eventStorage.InsertEvents(ctx, events); err != nil {
		return fmt.Errorf("failed to insert events: %w", err)
	}
	
	// Update contract's current block
	if err := i.contractStorage.UpdateContractBlock(ctx, contract.Address, toBlock); err != nil {
		return fmt.Errorf("failed to update contract block: %w", err)
	}
	
	// Update indexer state
	if err := i.stateStorage.UpdateLastIndexedBlock(
		ctx,
		contract.Address,
		toBlock,
		models.Hash(block.Hash().Hex()),
	); err != nil {
		return fmt.Errorf("failed to update indexer state: %w", err)
	}
	
	// Reset error count on success
	if err := i.stateStorage.ResetErrorCount(ctx, contract.Address); err != nil {
		i.logger.WithError(err).Warn("Failed to reset error count")
	}
	
	i.logger.WithFields(map[string]interface{}{
		"contract":      contract.Address,
		"from_block":    fromBlock,
		"to_block":      toBlock,
		"events_found":  len(events),
		"logs_found":    len(logs),
	}).Info("Successfully processed contract")
	
	return nil
}

// AddContract adds a new contract to monitor
func (i *Indexer) AddContract(ctx context.Context, contract *models.Contract) error {
	// Validate the contract
	if err := contract.Validate(); err != nil {
		return fmt.Errorf("invalid contract: %w", err)
	}
	
	// Create parser to validate ABI
	if err := i.createParserForContract(contract); err != nil {
		return fmt.Errorf("failed to create parser: %w", err)
	}
	
	// Save contract to database
	if err := i.contractStorage.UpsertContract(ctx, contract); err != nil {
		return fmt.Errorf("failed to save contract: %w", err)
	}
	
	// Initialize indexer state
	if err := i.stateStorage.InitializeState(ctx, contract.Address, contract.StartBlock); err != nil {
		return fmt.Errorf("failed to initialize state: %w", err)
	}
	
	i.logger.WithFields(map[string]interface{}{
		"contract": contract.Address,
		"name":     contract.Name,
	}).Info("Contract added to indexer")
	
	return nil
}

// RemoveContract removes a contract from monitoring
func (i *Indexer) RemoveContract(ctx context.Context, address models.Address) error {
	// Remove parser
	i.parsersMu.Lock()
	delete(i.parsers, address)
	i.parsersMu.Unlock()
	
	// Delete contract from database
	if err := i.contractStorage.DeleteContract(ctx, address); err != nil {
		return fmt.Errorf("failed to delete contract: %w", err)
	}
	
	// Delete indexer state
	if err := i.stateStorage.DeleteIndexerState(ctx, address); err != nil {
		i.logger.WithError(err).Warn("Failed to delete indexer state")
	}
	
	i.logger.WithField("contract", address).Info("Contract removed from indexer")
	
	return nil
}

// GetStats returns indexing statistics
func (i *Indexer) GetStats(ctx context.Context) (map[string]interface{}, error) {
	contractCount, err := i.contractStorage.GetContractCount(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get contract count: %w", err)
	}
	
	eventCount, err := i.eventStorage.GetEventCount(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get event count: %w", err)
	}
	
	latestBlock, err := i.client.GetLatestBlockNumber(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest block: %w", err)
	}
	
	stats := map[string]interface{}{
		"contracts_monitored": contractCount,
		"events_indexed":      eventCount,
		"latest_block":        latestBlock,
		"poll_interval":       i.pollInterval.String(),
		"batch_size":          i.batchSize,
	}
	
	return stats, nil
}

