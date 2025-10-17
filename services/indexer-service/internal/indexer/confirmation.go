package indexer

import (
	"context"
	"fmt"

	"github.com/smart-contract-event-indexer/shared/models"
	"github.com/smart-contract-event-indexer/shared/utils"
)

// ConfirmationChecker checks if blocks have sufficient confirmations
type ConfirmationChecker struct {
	logger utils.Logger
}

// NewConfirmationChecker creates a new confirmation checker
func NewConfirmationChecker(logger utils.Logger) *ConfirmationChecker {
	return &ConfirmationChecker{
		logger: logger,
	}
}

// IsBlockConfirmed checks if a block has sufficient confirmations
func (c *ConfirmationChecker) IsBlockConfirmed(blockNumber, latestBlock int64, requiredConfirmations int) bool {
	confirmations := latestBlock - blockNumber
	return confirmations >= int64(requiredConfirmations)
}

// GetConfirmedBlock returns the highest confirmed block number
func (c *ConfirmationChecker) GetConfirmedBlock(latestBlock int64, requiredConfirmations int) int64 {
	return latestBlock - int64(requiredConfirmations)
}

// GetConfirmationCount returns the number of confirmations for a block
func (c *ConfirmationChecker) GetConfirmationCount(blockNumber, latestBlock int64) int64 {
	return latestBlock - blockNumber
}

// WaitForConfirmation calculates how many blocks to wait for confirmation
func (c *ConfirmationChecker) WaitForConfirmation(blockNumber, latestBlock int64, requiredConfirmations int) int64 {
	currentConfirmations := latestBlock - blockNumber
	remainingConfirmations := int64(requiredConfirmations) - currentConfirmations
	
	if remainingConfirmations <= 0 {
		return 0
	}
	
	return remainingConfirmations
}

// ValidateConfirmationStrategy validates a confirmation strategy
func (c *ConfirmationChecker) ValidateConfirmationStrategy(strategy models.ConfirmationStrategy) error {
	switch strategy {
	case models.StrategyRealtime, models.StrategyBalanced, models.StrategySafe:
		return nil
	default:
		return fmt.Errorf("invalid confirmation strategy: %s", strategy)
	}
}

// GetStrategyDescription returns a human-readable description of a strategy
func (c *ConfirmationChecker) GetStrategyDescription(strategy models.ConfirmationStrategy) string {
	switch strategy {
	case models.StrategyRealtime:
		return "Realtime (1 block, ~12s delay, higher reorg risk)"
	case models.StrategyBalanced:
		return "Balanced (6 blocks, ~72s delay, recommended)"
	case models.StrategySafe:
		return "Safe (12 blocks, ~144s delay, lowest reorg risk)"
	default:
		return "Unknown strategy"
	}
}

// CalculateIndexingDelay calculates the expected indexing delay for a strategy
func (c *ConfirmationChecker) CalculateIndexingDelay(strategy models.ConfirmationStrategy, avgBlockTime int) int {
	blocks := strategy.ToBlocks()
	return blocks * avgBlockTime
}

// RecommendStrategy recommends a confirmation strategy based on requirements
func (c *ConfirmationChecker) RecommendStrategy(prioritizeSpeed bool, tolerateReorg bool) models.ConfirmationStrategy {
	if prioritizeSpeed && tolerateReorg {
		c.logger.Info("Recommending Realtime strategy (speed priority)")
		return models.StrategyRealtime
	}
	
	if !prioritizeSpeed && !tolerateReorg {
		c.logger.Info("Recommending Safe strategy (security priority)")
		return models.StrategySafe
	}
	
	c.logger.Info("Recommending Balanced strategy (default)")
	return models.StrategyBalanced
}

// GetConfirmationStatus returns detailed confirmation status for a block
func (c *ConfirmationChecker) GetConfirmationStatus(
	blockNumber,
	latestBlock int64,
	contract *models.Contract,
) *ConfirmationStatus {
	confirmations := latestBlock - blockNumber
	required := int64(contract.ConfirmBlocks)
	
	return &ConfirmationStatus{
		BlockNumber:            blockNumber,
		LatestBlock:            latestBlock,
		CurrentConfirmations:   confirmations,
		RequiredConfirmations:  required,
		IsConfirmed:            confirmations >= required,
		RemainingConfirmations: max(0, required-confirmations),
		ConfirmationProgress:   float64(confirmations) / float64(required) * 100,
	}
}

// ConfirmationStatus represents the confirmation status of a block
type ConfirmationStatus struct {
	BlockNumber            int64   `json:"blockNumber"`
	LatestBlock            int64   `json:"latestBlock"`
	CurrentConfirmations   int64   `json:"currentConfirmations"`
	RequiredConfirmations  int64   `json:"requiredConfirmations"`
	IsConfirmed            bool    `json:"isConfirmed"`
	RemainingConfirmations int64   `json:"remainingConfirmations"`
	ConfirmationProgress   float64 `json:"confirmationProgress"` // Percentage (0-100)
}

// Helper function
func max(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

// BatchConfirmationCheck checks confirmations for multiple blocks
type BatchConfirmationCheck struct {
	Blocks         []int64
	LatestBlock    int64
	Confirmations  int
	Results        map[int64]bool // blockNumber -> isConfirmed
}

// CheckBatch checks confirmations for a batch of blocks
func (c *ConfirmationChecker) CheckBatch(ctx context.Context, check *BatchConfirmationCheck) error {
	check.Results = make(map[int64]bool)
	
	for _, blockNumber := range check.Blocks {
		isConfirmed := c.IsBlockConfirmed(blockNumber, check.LatestBlock, check.Confirmations)
		check.Results[blockNumber] = isConfirmed
	}
	
	c.logger.WithField("checked_blocks", len(check.Blocks)).Debug("Batch confirmation check completed")
	
	return nil
}

// GetUnconfirmedBlocks returns blocks that don't have sufficient confirmations
func (c *ConfirmationChecker) GetUnconfirmedBlocks(blocks []int64, latestBlock int64, confirmations int) []int64 {
	unconfirmed := make([]int64, 0)
	
	for _, blockNumber := range blocks {
		if !c.IsBlockConfirmed(blockNumber, latestBlock, confirmations) {
			unconfirmed = append(unconfirmed, blockNumber)
		}
	}
	
	return unconfirmed
}

