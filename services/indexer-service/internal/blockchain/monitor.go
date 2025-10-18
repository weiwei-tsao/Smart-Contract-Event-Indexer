package blockchain

import (
	"context"
	"fmt"
	"time"

	"github.com/smart-contract-event-indexer/shared/utils"
)

// BlockMonitor monitors the blockchain for new blocks
type BlockMonitor struct {
	client       *Client
	pollInterval time.Duration
	logger       utils.Logger
	lastBlock    int64
}

// NewBlockMonitor creates a new block monitor
func NewBlockMonitor(client *Client, pollInterval time.Duration, logger utils.Logger) *BlockMonitor {
	return &BlockMonitor{
		client:       client,
		pollInterval: pollInterval,
		logger:       logger,
		lastBlock:    0,
	}
}

// Start begins monitoring for new blocks
func (m *BlockMonitor) Start(ctx context.Context, blockChan chan<- int64) error {
	m.logger.WithField("poll_interval", m.pollInterval).Info("Starting block monitor")
	
	// Get the initial block number
	latestBlock, err := m.client.GetLatestBlockNumber(ctx)
	if err != nil {
		return fmt.Errorf("failed to get initial block number: %w", err)
	}
	
	m.lastBlock = latestBlock
	m.logger.WithField("initial_block", latestBlock).Info("Block monitor initialized")
	
	// Start polling loop
	ticker := time.NewTicker(m.pollInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			m.logger.Info("Block monitor stopping")
			return ctx.Err()
			
		case <-ticker.C:
			if err := m.checkForNewBlocks(ctx, blockChan); err != nil {
				m.logger.WithError(err).Error("Error checking for new blocks")
				// Continue polling despite errors
			}
		}
	}
}

// checkForNewBlocks checks for new blocks and sends them to the channel
func (m *BlockMonitor) checkForNewBlocks(ctx context.Context, blockChan chan<- int64) error {
	latestBlock, err := m.client.GetLatestBlockNumber(ctx)
	if err != nil {
		return fmt.Errorf("failed to get latest block number: %w", err)
	}
	
	// If there are new blocks, send them to the channel
	if latestBlock > m.lastBlock {
		newBlocks := latestBlock - m.lastBlock
		m.logger.WithFields(map[string]interface{}{
			"last_block":   m.lastBlock,
			"latest_block": latestBlock,
			"new_blocks":   newBlocks,
		}).Debug("New blocks detected")
		
		// Send the latest block number to the channel
		select {
		case blockChan <- latestBlock:
			m.lastBlock = latestBlock
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	
	return nil
}

// GetLastBlock returns the last block number seen
func (m *BlockMonitor) GetLastBlock() int64 {
	return m.lastBlock
}

// SetLastBlock sets the last block number (useful for resuming from a saved state)
func (m *BlockMonitor) SetLastBlock(blockNumber int64) {
	m.lastBlock = blockNumber
	m.logger.WithField("block_number", blockNumber).Info("Last block set")
}

