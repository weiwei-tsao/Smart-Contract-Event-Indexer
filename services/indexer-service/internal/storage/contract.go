package storage

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/smart-contract-event-indexer/shared/models"
	"github.com/smart-contract-event-indexer/shared/utils"
)

// ContractStorage handles database operations for contracts
type ContractStorage struct {
	db     *sqlx.DB
	logger utils.Logger
}

// NewContractStorage creates a new contract storage
func NewContractStorage(db *sqlx.DB, logger utils.Logger) *ContractStorage {
	return &ContractStorage{
		db:     db,
		logger: logger,
	}
}

// GetContract retrieves a contract by address
func (s *ContractStorage) GetContract(ctx context.Context, address models.Address) (*models.Contract, error) {
	var contract models.Contract
	
	query := `
		SELECT id, address, abi, name, start_block, current_block, confirm_blocks, created_at, updated_at
		FROM contracts
		WHERE address = $1
	`
	
	err := s.db.GetContext(ctx, &contract, query, address)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("contract %s not found", address)
		}
		return nil, fmt.Errorf("failed to get contract: %w", err)
	}
	
	return &contract, nil
}

// GetAllContracts retrieves all monitored contracts
func (s *ContractStorage) GetAllContracts(ctx context.Context) ([]*models.Contract, error) {
	var contracts []*models.Contract
	
	query := `
		SELECT id, address, abi, name, start_block, current_block, confirm_blocks, created_at, updated_at
		FROM contracts
		ORDER BY created_at ASC
	`
	
	err := s.db.SelectContext(ctx, &contracts, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get all contracts: %w", err)
	}
	
	s.logger.WithField("count", len(contracts)).Debug("Retrieved all contracts")
	
	return contracts, nil
}

// CreateContract inserts a new contract
func (s *ContractStorage) CreateContract(ctx context.Context, contract *models.Contract) error {
	query := `
		INSERT INTO contracts (address, abi, name, start_block, current_block, confirm_blocks)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at
	`
	
	err := s.db.QueryRowContext(
		ctx,
		query,
		contract.Address,
		contract.ABI,
		contract.Name,
		contract.StartBlock,
		contract.CurrentBlock,
		contract.ConfirmBlocks,
	).Scan(&contract.ID, &contract.CreatedAt, &contract.UpdatedAt)
	
	if err != nil {
		return fmt.Errorf("failed to create contract: %w", err)
	}
	
	s.logger.WithFields(map[string]interface{}{
		"address": contract.Address,
		"name":    contract.Name,
	}).Info("Contract created")
	
	return nil
}

// UpdateContractBlock updates the current block for a contract
func (s *ContractStorage) UpdateContractBlock(ctx context.Context, address models.Address, blockNumber int64) error {
	query := `
		UPDATE contracts
		SET current_block = $1, updated_at = NOW()
		WHERE address = $2
	`
	
	result, err := s.db.ExecContext(ctx, query, blockNumber, address)
	if err != nil {
		return fmt.Errorf("failed to update contract block: %w", err)
	}
	
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	if rows == 0 {
		return fmt.Errorf("contract %s not found", address)
	}
	
	s.logger.WithFields(map[string]interface{}{
		"address": address,
		"block":   blockNumber,
	}).Debug("Contract block updated")
	
	return nil
}

// DeleteContract removes a contract from monitoring
func (s *ContractStorage) DeleteContract(ctx context.Context, address models.Address) error {
	query := `DELETE FROM contracts WHERE address = $1`
	
	result, err := s.db.ExecContext(ctx, query, address)
	if err != nil {
		return fmt.Errorf("failed to delete contract: %w", err)
	}
	
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	if rows == 0 {
		return fmt.Errorf("contract %s not found", address)
	}
	
	s.logger.WithField("address", address).Info("Contract deleted")
	
	return nil
}

// UpsertContract inserts or updates a contract (idempotent)
func (s *ContractStorage) UpsertContract(ctx context.Context, contract *models.Contract) error {
	query := `
		INSERT INTO contracts (address, abi, name, start_block, current_block, confirm_blocks)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (address) DO UPDATE
		SET abi = EXCLUDED.abi,
		    name = EXCLUDED.name,
		    start_block = EXCLUDED.start_block,
		    confirm_blocks = EXCLUDED.confirm_blocks,
		    updated_at = NOW()
		RETURNING id, created_at, updated_at
	`
	
	err := s.db.QueryRowContext(
		ctx,
		query,
		contract.Address,
		contract.ABI,
		contract.Name,
		contract.StartBlock,
		contract.CurrentBlock,
		contract.ConfirmBlocks,
	).Scan(&contract.ID, &contract.CreatedAt, &contract.UpdatedAt)
	
	if err != nil {
		return fmt.Errorf("failed to upsert contract: %w", err)
	}
	
	s.logger.WithFields(map[string]interface{}{
		"address": contract.Address,
		"name":    contract.Name,
	}).Info("Contract upserted")
	
	return nil
}

// ContractExists checks if a contract exists
func (s *ContractStorage) ContractExists(ctx context.Context, address models.Address) (bool, error) {
	var exists bool
	
	query := `SELECT EXISTS(SELECT 1 FROM contracts WHERE address = $1)`
	
	err := s.db.GetContext(ctx, &exists, query, address)
	if err != nil {
		return false, fmt.Errorf("failed to check contract existence: %w", err)
	}
	
	return exists, nil
}

// GetContractCount returns the total number of monitored contracts
func (s *ContractStorage) GetContractCount(ctx context.Context) (int, error) {
	var count int
	
	query := `SELECT COUNT(*) FROM contracts`
	
	err := s.db.GetContext(ctx, &count, query)
	if err != nil {
		return 0, fmt.Errorf("failed to get contract count: %w", err)
	}
	
	return count, nil
}

