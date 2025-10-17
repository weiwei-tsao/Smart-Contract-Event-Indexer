package storage

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/smart-contract-event-indexer/shared/models"
	"github.com/smart-contract-event-indexer/shared/utils"
)

// StateStorage handles database operations for indexer state
type StateStorage struct {
	db     *sqlx.DB
	logger utils.Logger
}

// NewStateStorage creates a new state storage
func NewStateStorage(db *sqlx.DB, logger utils.Logger) *StateStorage {
	return &StateStorage{
		db:     db,
		logger: logger,
	}
}

// GetIndexerState retrieves the indexer state for a contract
func (s *StateStorage) GetIndexerState(ctx context.Context, contractAddress models.Address) (*models.IndexerState, error) {
	var state models.IndexerState
	
	query := `
		SELECT id, contract_address, last_indexed_block, last_block_hash,
		       last_processed_at, status, error_count, last_error, created_at, updated_at
		FROM indexer_state
		WHERE contract_address = $1
	`
	
	err := s.db.GetContext(ctx, &state, query, contractAddress)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("indexer state for contract %s not found", contractAddress)
		}
		return nil, fmt.Errorf("failed to get indexer state: %w", err)
	}
	
	return &state, nil
}

// SaveIndexerState saves or updates the indexer state
func (s *StateStorage) SaveIndexerState(ctx context.Context, state *models.IndexerState) error {
	query := `
		INSERT INTO indexer_state (
			contract_address, last_indexed_block, last_block_hash,
			last_processed_at, status, error_count, last_error
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (contract_address) DO UPDATE
		SET last_indexed_block = EXCLUDED.last_indexed_block,
		    last_block_hash = EXCLUDED.last_block_hash,
		    last_processed_at = EXCLUDED.last_processed_at,
		    status = EXCLUDED.status,
		    error_count = EXCLUDED.error_count,
		    last_error = EXCLUDED.last_error,
		    updated_at = NOW()
		RETURNING id, created_at, updated_at
	`
	
	err := s.db.QueryRowContext(
		ctx,
		query,
		state.ContractAddress,
		state.LastIndexedBlock,
		state.LastBlockHash,
		state.LastProcessedAt,
		state.Status,
		state.ErrorCount,
		state.LastError,
	).Scan(&state.ID, &state.CreatedAt, &state.UpdatedAt)
	
	if err != nil {
		return fmt.Errorf("failed to save indexer state: %w", err)
	}
	
	s.logger.WithFields(map[string]interface{}{
		"contract": state.ContractAddress,
		"block":    state.LastIndexedBlock,
		"status":   state.Status,
	}).Debug("Indexer state saved")
	
	return nil
}

// UpdateLastIndexedBlock updates only the last indexed block
func (s *StateStorage) UpdateLastIndexedBlock(ctx context.Context, contractAddress models.Address, blockNumber int64, blockHash models.Hash) error {
	query := `
		UPDATE indexer_state
		SET last_indexed_block = $1,
		    last_block_hash = $2,
		    last_processed_at = NOW(),
		    updated_at = NOW()
		WHERE contract_address = $3
	`
	
	result, err := s.db.ExecContext(ctx, query, blockNumber, blockHash, contractAddress)
	if err != nil {
		return fmt.Errorf("failed to update last indexed block: %w", err)
	}
	
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	if rows == 0 {
		// State doesn't exist, create it
		state := &models.IndexerState{
			ContractAddress:   contractAddress,
			LastIndexedBlock:  blockNumber,
			LastBlockHash:     blockHash,
			Status:            "active",
			ErrorCount:        0,
		}
		return s.SaveIndexerState(ctx, state)
	}
	
	return nil
}

// IncrementErrorCount increments the error count for a contract
func (s *StateStorage) IncrementErrorCount(ctx context.Context, contractAddress models.Address, errorMessage string) error {
	query := `
		UPDATE indexer_state
		SET error_count = error_count + 1,
		    last_error = $1,
		    updated_at = NOW()
		WHERE contract_address = $2
	`
	
	_, err := s.db.ExecContext(ctx, query, errorMessage, contractAddress)
	if err != nil {
		return fmt.Errorf("failed to increment error count: %w", err)
	}
	
	return nil
}

// ResetErrorCount resets the error count for a contract
func (s *StateStorage) ResetErrorCount(ctx context.Context, contractAddress models.Address) error {
	query := `
		UPDATE indexer_state
		SET error_count = 0,
		    last_error = NULL,
		    updated_at = NOW()
		WHERE contract_address = $1
	`
	
	_, err := s.db.ExecContext(ctx, query, contractAddress)
	if err != nil {
		return fmt.Errorf("failed to reset error count: %w", err)
	}
	
	return nil
}

// UpdateStatus updates the indexer status for a contract
func (s *StateStorage) UpdateStatus(ctx context.Context, contractAddress models.Address, status string) error {
	query := `
		UPDATE indexer_state
		SET status = $1,
		    updated_at = NOW()
		WHERE contract_address = $2
	`
	
	_, err := s.db.ExecContext(ctx, query, status, contractAddress)
	if err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}
	
	s.logger.WithFields(map[string]interface{}{
		"contract": contractAddress,
		"status":   status,
	}).Info("Indexer status updated")
	
	return nil
}

// GetAllIndexerStates retrieves all indexer states
func (s *StateStorage) GetAllIndexerStates(ctx context.Context) ([]*models.IndexerState, error) {
	var states []*models.IndexerState
	
	query := `
		SELECT id, contract_address, last_indexed_block, last_block_hash,
		       last_processed_at, status, error_count, last_error, created_at, updated_at
		FROM indexer_state
		ORDER BY contract_address ASC
	`
	
	err := s.db.SelectContext(ctx, &states, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get all indexer states: %w", err)
	}
	
	return states, nil
}

// DeleteIndexerState deletes the indexer state for a contract
func (s *StateStorage) DeleteIndexerState(ctx context.Context, contractAddress models.Address) error {
	query := `DELETE FROM indexer_state WHERE contract_address = $1`
	
	_, err := s.db.ExecContext(ctx, query, contractAddress)
	if err != nil {
		return fmt.Errorf("failed to delete indexer state: %w", err)
	}
	
	s.logger.WithField("contract", contractAddress).Info("Indexer state deleted")
	
	return nil
}

// InitializeState creates initial state for a contract if it doesn't exist
func (s *StateStorage) InitializeState(ctx context.Context, contractAddress models.Address, startBlock int64) error {
	query := `
		INSERT INTO indexer_state (
			contract_address, last_indexed_block, status, error_count
		)
		VALUES ($1, $2, 'active', 0)
		ON CONFLICT (contract_address) DO NOTHING
	`
	
	_, err := s.db.ExecContext(ctx, query, contractAddress, startBlock-1)
	if err != nil {
		return fmt.Errorf("failed to initialize state: %w", err)
	}
	
	return nil
}

