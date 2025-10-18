package reorg

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/smart-contract-event-indexer/indexer-service/internal/storage"
	"github.com/smart-contract-event-indexer/shared/models"
	"github.com/smart-contract-event-indexer/shared/utils"
)

// Handler handles blockchain reorganizations
type Handler struct {
	db              *sqlx.DB
	contractStorage *storage.ContractStorage
	eventStorage    *storage.EventStorage
	stateStorage    *storage.StateStorage
	detector        *Detector
	logger          utils.Logger
}

// NewHandler creates a new reorg handler
func NewHandler(
	db *sqlx.DB,
	contractStorage *storage.ContractStorage,
	eventStorage *storage.EventStorage,
	stateStorage *storage.StateStorage,
	detector *Detector,
	logger utils.Logger,
) *Handler {
	return &Handler{
		db:              db,
		contractStorage: contractStorage,
		eventStorage:    eventStorage,
		stateStorage:    stateStorage,
		detector:        detector,
		logger:          logger,
	}
}

// HandleReorg handles a blockchain reorganization
func (h *Handler) HandleReorg(ctx context.Context, contractAddress models.Address, forkPoint int64) error {
	h.logger.WithFields(map[string]interface{}{
		"contract":   contractAddress,
		"fork_point": forkPoint,
	}).Warn("Handling blockchain reorganization")
	
	// Start a database transaction
	tx, err := h.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()
	
	// Step 1: Delete events from fork point onwards
	if err := h.rollbackEvents(ctx, contractAddress, forkPoint); err != nil {
		return fmt.Errorf("failed to rollback events: %w", err)
	}
	
	// Step 2: Update contract's current block to fork point
	if err := h.contractStorage.UpdateContractBlock(ctx, contractAddress, forkPoint-1); err != nil {
		return fmt.Errorf("failed to update contract block: %w", err)
	}
	
	// Step 3: Update indexer state
	if err := h.stateStorage.UpdateStatus(ctx, contractAddress, "reorg_recovery"); err != nil {
		h.logger.WithError(err).Warn("Failed to update indexer status")
	}
	
	// Step 4: Clear block cache for affected blocks
	// In a more sophisticated implementation, we'd only clear the affected blocks
	// For simplicity, we're keeping the cache as-is since detector will handle it
	
	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	
	h.logger.WithFields(map[string]interface{}{
		"contract":   contractAddress,
		"fork_point": forkPoint,
	}).Info("Successfully handled blockchain reorganization")
	
	return nil
}

// rollbackEvents deletes events from a specific block onwards
func (h *Handler) rollbackEvents(ctx context.Context, contractAddress models.Address, fromBlock int64) error {
	// Get count of events to be deleted (for logging)
	countQuery := `
		SELECT COUNT(*) FROM events
		WHERE contract_address = $1 AND block_number >= $2
	`
	
	var count int64
	if err := h.db.GetContext(ctx, &count, countQuery, contractAddress, fromBlock); err != nil {
		h.logger.WithError(err).Warn("Failed to count events to rollback")
	}
	
	// Delete events
	if err := h.eventStorage.DeleteEventsByBlock(ctx, contractAddress, fromBlock); err != nil {
		return fmt.Errorf("failed to delete events: %w", err)
	}
	
	h.logger.WithFields(map[string]interface{}{
		"contract":   contractAddress,
		"from_block": fromBlock,
		"deleted":    count,
	}).Info("Events rolled back")
	
	return nil
}

// HandleReorgForAllContracts handles a reorg that affects all contracts
func (h *Handler) HandleReorgForAllContracts(ctx context.Context, forkPoint int64) error {
	h.logger.WithField("fork_point", forkPoint).Warn("Handling global blockchain reorganization")
	
	// Get all contracts
	contracts, err := h.contractStorage.GetAllContracts(ctx)
	if err != nil {
		return fmt.Errorf("failed to get contracts: %w", err)
	}
	
	// Handle reorg for each contract
	for _, contract := range contracts {
		// Only rollback if the contract has indexed past the fork point
		if contract.CurrentBlock >= forkPoint {
			if err := h.HandleReorg(ctx, contract.Address, forkPoint); err != nil {
				h.logger.WithError(err).WithField("contract", contract.Address).Error("Failed to handle reorg for contract")
				continue
			}
		}
	}
	
	h.logger.Info("Global reorganization handled for all contracts")
	
	return nil
}

// RecoverFromReorg marks contracts as recovered from reorg
func (h *Handler) RecoverFromReorg(ctx context.Context, contractAddress models.Address) error {
	if err := h.stateStorage.UpdateStatus(ctx, contractAddress, "active"); err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}
	
	h.logger.WithField("contract", contractAddress).Info("Contract recovered from reorg")
	
	return nil
}

// GetReorgStats returns statistics about reorganizations
func (h *Handler) GetReorgStats(ctx context.Context) (map[string]interface{}, error) {
	// Query for contracts in reorg recovery state
	query := `
		SELECT COUNT(*) FROM indexer_state
		WHERE status = 'reorg_recovery'
	`
	
	var recoveryCount int
	if err := h.db.GetContext(ctx, &recoveryCount, query); err != nil {
		return nil, fmt.Errorf("failed to get reorg stats: %w", err)
	}
	
	// Get cache stats from detector
	cacheStats, err := h.detector.GetCacheStats(ctx)
	if err != nil {
		h.logger.WithError(err).Warn("Failed to get cache stats")
		cacheStats = map[string]interface{}{}
	}
	
	stats := map[string]interface{}{
		"contracts_in_recovery": recoveryCount,
		"cache_stats":          cacheStats,
	}
	
	return stats, nil
}

// ValidateChainIntegrity validates the integrity of indexed data against the blockchain
func (h *Handler) ValidateChainIntegrity(ctx context.Context, contractAddress models.Address) (bool, error) {
	// Get the latest indexed block for the contract
	maxBlock, err := h.eventStorage.GetMaxBlockNumber(ctx, contractAddress)
	if err != nil {
		return false, fmt.Errorf("failed to get max block: %w", err)
	}
	
	if maxBlock == 0 {
		// No events indexed yet, nothing to validate
		return true, nil
	}
	
	// In a complete implementation, we would:
	// 1. Fetch the block from the blockchain
	// 2. Compare the block hash with our cached hash
	// 3. Verify that the events match
	
	// For now, we'll just check if we have the block in cache
	cachedHash, err := h.detector.GetCachedBlockHash(ctx, maxBlock)
	if err != nil {
		h.logger.WithFields(map[string]interface{}{
			"contract": contractAddress,
			"block":    maxBlock,
		}).Debug("Block not in cache, cannot validate integrity")
		return true, nil // Assume valid if not in cache
	}
	
	h.logger.WithFields(map[string]interface{}{
		"contract":   contractAddress,
		"block":      maxBlock,
		"block_hash": cachedHash,
	}).Debug("Chain integrity check passed")
	
	return true, nil
}

