package indexer

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/smart-contract-event-indexer/shared/models"
	"github.com/smart-contract-event-indexer/shared/utils"
)

// LifecycleManager manages the lifecycle of the indexer
type LifecycleManager struct {
	indexer         *Indexer
	logger          *utils.Logger
	shutdownTimeout time.Duration
	
	// State tracking
	isRunning       bool
	mu              sync.RWMutex
	
	// Graceful shutdown
	activeJobs      sync.WaitGroup
	shutdownChan    chan struct{}
}

// NewLifecycleManager creates a new lifecycle manager
func NewLifecycleManager(indexer *Indexer, logger utils.Logger, shutdownTimeout time.Duration) *LifecycleManager {
	return &LifecycleManager{
		indexer:         indexer,
		logger:          logger,
		shutdownTimeout: shutdownTimeout,
		shutdownChan:    make(chan struct{}),
	}
}

// Start starts the indexer with lifecycle management
func (m *LifecycleManager) Start(ctx context.Context) error {
	m.mu.Lock()
	if m.isRunning {
		m.mu.Unlock()
		return fmt.Errorf("indexer is already running")
	}
	m.isRunning = true
	m.mu.Unlock()
	
	m.logger.Info("Lifecycle manager starting indexer")
	
	// Load state from database
	if err := m.recoverState(ctx); err != nil {
		m.logger.WithError(err).Warn("Failed to recover state, starting fresh")
	}
	
	// Start the indexer
	if err := m.indexer.Start(ctx); err != nil {
		m.mu.Lock()
		m.isRunning = false
		m.mu.Unlock()
		return err
	}
	
	return nil
}

// Stop gracefully stops the indexer
func (m *LifecycleManager) Stop(ctx context.Context) error {
	m.mu.Lock()
	if !m.isRunning {
		m.mu.Unlock()
		return fmt.Errorf("indexer is not running")
	}
	m.mu.Unlock()
	
	m.logger.Info("Gracefully stopping indexer")
	
	// Signal shutdown
	close(m.shutdownChan)
	
	// Wait for active jobs to complete with timeout
	done := make(chan struct{})
	go func() {
		m.activeJobs.Wait()
		close(done)
	}()
	
	select {
	case <-done:
		m.logger.Info("All active jobs completed")
	case <-time.After(m.shutdownTimeout):
		m.logger.Warn("Shutdown timeout reached, some jobs may be incomplete")
	case <-ctx.Done():
		m.logger.Warn("Shutdown context cancelled")
	}
	
	// Save final state
	if err := m.saveState(ctx); err != nil {
		m.logger.WithError(err).Error("Failed to save state during shutdown")
	}
	
	m.mu.Lock()
	m.isRunning = false
	m.mu.Unlock()
	
	m.logger.Info("Indexer stopped successfully")
	
	return nil
}

// IsRunning returns whether the indexer is running
func (m *LifecycleManager) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.isRunning
}

// recoverState recovers the indexer state from the database
func (m *LifecycleManager) recoverState(ctx context.Context) error {
	m.logger.Info("Recovering indexer state from database")
	
	// Get all indexer states
	states, err := m.indexer.stateStorage.GetAllIndexerStates(ctx)
	if err != nil {
		return fmt.Errorf("failed to get indexer states: %w", err)
	}
	
	if len(states) == 0 {
		m.logger.Info("No previous state found, starting fresh")
		return nil
	}
	
	// Log recovery information
	for _, state := range states {
		m.logger.WithFields(map[string]interface{}{
			"contract":      state.ContractAddress,
			"last_block":    state.LastIndexedBlock,
			"status":        state.Status,
			"error_count":   state.ErrorCount,
			"last_processed": state.LastProcessedAt,
		}).Info("Recovered contract state")
		
		// Reset status from reorg_recovery to active if needed
		if state.Status == "reorg_recovery" {
			if err := m.indexer.stateStorage.UpdateStatus(ctx, state.ContractAddress, "active"); err != nil {
				m.logger.WithError(err).Warn("Failed to reset reorg_recovery status")
			}
		}
	}
	
	m.logger.WithField("contract_count", len(states)).Info("State recovery completed")
	
	return nil
}

// saveState saves the current indexer state to the database
func (m *LifecycleManager) saveState(ctx context.Context) error {
	m.logger.Info("Saving indexer state")
	
	// Get all contracts
	contracts, err := m.indexer.contractStorage.GetAllContracts(ctx)
	if err != nil {
		return fmt.Errorf("failed to get contracts: %w", err)
	}
	
	// Save state for each contract
	for _, contract := range contracts {
		state := &models.IndexerState{
			ContractAddress:   contract.Address,
			LastIndexedBlock:  contract.CurrentBlock,
			LastProcessedAt:   time.Now().UTC(),
			Status:            "stopped",
		}
		
		if err := m.indexer.stateStorage.SaveIndexerState(ctx, state); err != nil {
			m.logger.WithError(err).WithField("contract", contract.Address).Error("Failed to save contract state")
			continue
		}
	}
	
	m.logger.Info("Indexer state saved successfully")
	
	return nil
}

// TrackJob tracks an active job for graceful shutdown
func (m *LifecycleManager) TrackJob() func() {
	m.activeJobs.Add(1)
	return func() {
		m.activeJobs.Done()
	}
}

// ShouldShutdown returns whether a shutdown has been requested
func (m *LifecycleManager) ShouldShutdown() bool {
	select {
	case <-m.shutdownChan:
		return true
	default:
		return false
	}
}

// WaitForShutdown blocks until shutdown is requested
func (m *LifecycleManager) WaitForShutdown() {
	<-m.shutdownChan
}

// GetStatus returns the current lifecycle status
func (m *LifecycleManager) GetStatus() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	return map[string]interface{}{
		"is_running":        m.isRunning,
		"shutdown_timeout":  m.shutdownTimeout.String(),
	}
}

// Restart restarts the indexer
func (m *LifecycleManager) Restart(ctx context.Context) error {
	m.logger.Info("Restarting indexer")
	
	// Stop the indexer
	if err := m.Stop(ctx); err != nil {
		return fmt.Errorf("failed to stop indexer: %w", err)
	}
	
	// Wait a bit before restarting
	time.Sleep(2 * time.Second)
	
	// Create new shutdown channel
	m.shutdownChan = make(chan struct{})
	
	// Start the indexer
	if err := m.Start(ctx); err != nil {
		return fmt.Errorf("failed to start indexer: %w", err)
	}
	
	m.logger.Info("Indexer restarted successfully")
	
	return nil
}

// HealthCheck performs a health check on the indexer
func (m *LifecycleManager) HealthCheck(ctx context.Context) error {
	m.mu.RLock()
	isRunning := m.isRunning
	m.mu.RUnlock()
	
	if !isRunning {
		return fmt.Errorf("indexer is not running")
	}
	
	// Check if we can get stats (indicates the indexer is responsive)
	_, err := m.indexer.GetStats(ctx)
	if err != nil {
		return fmt.Errorf("indexer health check failed: %w", err)
	}
	
	return nil
}

// Pause pauses indexing for a specific contract
func (m *LifecycleManager) Pause(ctx context.Context, contractAddress models.Address) error {
	if err := m.indexer.stateStorage.UpdateStatus(ctx, contractAddress, "paused"); err != nil {
		return fmt.Errorf("failed to pause contract: %w", err)
	}
	
	m.logger.WithField("contract", contractAddress).Info("Contract indexing paused")
	
	return nil
}

// Resume resumes indexing for a specific contract
func (m *LifecycleManager) Resume(ctx context.Context, contractAddress models.Address) error {
	if err := m.indexer.stateStorage.UpdateStatus(ctx, contractAddress, "active"); err != nil {
		return fmt.Errorf("failed to resume contract: %w", err)
	}
	
	m.logger.WithField("contract", contractAddress).Info("Contract indexing resumed")
	
	return nil
}

// GetUptime returns how long the indexer has been running
func (m *LifecycleManager) GetUptime() time.Duration {
	// This is a simplified version
	// In a real implementation, you'd track the start time
	return 0
}

