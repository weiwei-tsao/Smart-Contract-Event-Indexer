package storage

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/smart-contract-event-indexer/shared/models"
	"github.com/smart-contract-event-indexer/shared/utils"
)

// EventStorage handles database operations for events
type EventStorage struct {
	db     *sqlx.DB
	logger utils.Logger
}

// NewEventStorage creates a new event storage
func NewEventStorage(db *sqlx.DB, logger utils.Logger) *EventStorage {
	return &EventStorage{
		db:     db,
		logger: logger,
	}
}

// InsertEvent inserts a single event
func (s *EventStorage) InsertEvent(ctx context.Context, event *models.Event) error {
	query := `
		INSERT INTO events (
			contract_address, event_name, block_number, block_hash,
			transaction_hash, transaction_index, log_index, args, timestamp
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (transaction_hash, log_index) DO NOTHING
		RETURNING id, created_at
	`
	
	err := s.db.QueryRowContext(
		ctx,
		query,
		event.ContractAddress,
		event.EventName,
		event.BlockNumber,
		event.BlockHash,
		event.TransactionHash,
		event.TransactionIndex,
		event.LogIndex,
		event.Args,
		event.Timestamp,
	).Scan(&event.ID, &event.CreatedAt)
	
	if err != nil {
		// If it's a "no rows" error, it means ON CONFLICT triggered
		if strings.Contains(err.Error(), "no rows") {
			s.logger.Debug("Event already exists, skipping")
			return nil
		}
		return fmt.Errorf("failed to insert event: %w", err)
	}
	
	s.logger.WithFields(map[string]interface{}{
		"event_name": event.EventName,
		"block":      event.BlockNumber,
		"tx":         event.TransactionHash,
	}).Debug("Event inserted")
	
	return nil
}

// InsertEvents inserts multiple events in a batch
func (s *EventStorage) InsertEvents(ctx context.Context, events []*models.Event) error {
	if len(events) == 0 {
		return nil
	}
	
	// Start a transaction
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()
	
	// Prepare the query
	query := `
		INSERT INTO events (
			contract_address, event_name, block_number, block_hash,
			transaction_hash, transaction_index, log_index, args, timestamp
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (transaction_hash, log_index) DO NOTHING
	`
	
	stmt, err := tx.PreparexContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()
	
	// Insert each event
	insertedCount := 0
	for _, event := range events {
		result, err := stmt.ExecContext(
			ctx,
			event.ContractAddress,
			event.EventName,
			event.BlockNumber,
			event.BlockHash,
			event.TransactionHash,
			event.TransactionIndex,
			event.LogIndex,
			event.Args,
			event.Timestamp,
		)
		if err != nil {
			return fmt.Errorf("failed to insert event: %w", err)
		}
		
		rows, _ := result.RowsAffected()
		insertedCount += int(rows)
	}
	
	// Commit the transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	
	s.logger.WithFields(map[string]interface{}{
		"total":    len(events),
		"inserted": insertedCount,
		"skipped":  len(events) - insertedCount,
	}).Info("Events batch inserted")
	
	return nil
}

// GetEventsByContract retrieves events for a contract within a block range
func (s *EventStorage) GetEventsByContract(ctx context.Context, contractAddress models.Address, fromBlock, toBlock int64, limit int) ([]*models.Event, error) {
	var events []*models.Event
	
	query := `
		SELECT id, contract_address, event_name, block_number, block_hash,
		       transaction_hash, transaction_index, log_index, args, timestamp, created_at
		FROM events
		WHERE contract_address = $1
		  AND block_number >= $2
		  AND block_number <= $3
		ORDER BY block_number DESC, log_index DESC
		LIMIT $4
	`
	
	err := s.db.SelectContext(ctx, &events, query, contractAddress, fromBlock, toBlock, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get events by contract: %w", err)
	}
	
	return events, nil
}

// GetEventsByTransaction retrieves events for a specific transaction
func (s *EventStorage) GetEventsByTransaction(ctx context.Context, txHash models.Hash) ([]*models.Event, error) {
	var events []*models.Event
	
	query := `
		SELECT id, contract_address, event_name, block_number, block_hash,
		       transaction_hash, transaction_index, log_index, args, timestamp, created_at
		FROM events
		WHERE transaction_hash = $1
		ORDER BY log_index ASC
	`
	
	err := s.db.SelectContext(ctx, &events, query, txHash)
	if err != nil {
		return nil, fmt.Errorf("failed to get events by transaction: %w", err)
	}
	
	return events, nil
}

// GetLatestEvents retrieves the most recent events
func (s *EventStorage) GetLatestEvents(ctx context.Context, limit int) ([]*models.Event, error) {
	var events []*models.Event
	
	query := `
		SELECT id, contract_address, event_name, block_number, block_hash,
		       transaction_hash, transaction_index, log_index, args, timestamp, created_at
		FROM events
		ORDER BY block_number DESC, log_index DESC
		LIMIT $1
	`
	
	err := s.db.SelectContext(ctx, &events, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest events: %w", err)
	}
	
	return events, nil
}

// GetEventCount returns the total number of indexed events
func (s *EventStorage) GetEventCount(ctx context.Context) (int64, error) {
	var count int64
	
	query := `SELECT COUNT(*) FROM events`
	
	err := s.db.GetContext(ctx, &count, query)
	if err != nil {
		return 0, fmt.Errorf("failed to get event count: %w", err)
	}
	
	return count, nil
}

// GetEventCountByContract returns the number of events for a specific contract
func (s *EventStorage) GetEventCountByContract(ctx context.Context, contractAddress models.Address) (int64, error) {
	var count int64
	
	query := `SELECT COUNT(*) FROM events WHERE contract_address = $1`
	
	err := s.db.GetContext(ctx, &count, query, contractAddress)
	if err != nil {
		return 0, fmt.Errorf("failed to get event count by contract: %w", err)
	}
	
	return count, nil
}

// DeleteEventsByBlock deletes events from a specific block onwards (for reorg handling)
func (s *EventStorage) DeleteEventsByBlock(ctx context.Context, contractAddress models.Address, fromBlock int64) error {
	query := `
		DELETE FROM events
		WHERE contract_address = $1 AND block_number >= $2
	`
	
	result, err := s.db.ExecContext(ctx, query, contractAddress, fromBlock)
	if err != nil {
		return fmt.Errorf("failed to delete events: %w", err)
	}
	
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	s.logger.WithFields(map[string]interface{}{
		"contract":   contractAddress,
		"from_block": fromBlock,
		"deleted":    rows,
	}).Info("Events deleted for reorg")
	
	return nil
}

// GetMaxBlockNumber returns the highest block number for a contract
func (s *EventStorage) GetMaxBlockNumber(ctx context.Context, contractAddress models.Address) (int64, error) {
	var maxBlock sql.NullInt64
	
	query := `
		SELECT MAX(block_number)
		FROM events
		WHERE contract_address = $1
	`
	
	err := s.db.GetContext(ctx, &maxBlock, query, contractAddress)
	if err != nil {
		return 0, fmt.Errorf("failed to get max block number: %w", err)
	}
	
	if !maxBlock.Valid {
		return 0, nil
	}
	
	return maxBlock.Int64, nil
}

