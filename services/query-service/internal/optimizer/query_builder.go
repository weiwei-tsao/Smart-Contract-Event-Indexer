package optimizer

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/smart-contract-event-indexer/query-service/internal/service"
	"github.com/smart-contract-event-indexer/shared/models"
	"go.uber.org/zap"
)

// QueryBuilder handles SQL query construction and optimization
type QueryBuilder struct {
	db     *sql.DB
	logger *zap.Logger
}

// NewQueryBuilder creates a new QueryBuilder
func NewQueryBuilder(db *sql.DB, logger *zap.Logger) *QueryBuilder {
	return &QueryBuilder{
		db:     db,
		logger: logger,
	}
}

// BuildEventQuery builds and executes a query for events
func (qb *QueryBuilder) BuildEventQuery(ctx context.Context, query *service.EventQuery) ([]*models.Event, int32, error) {
	// Build the base query
	baseQuery := `
		SELECT 
			e.id, e.contract_id, e.contract_address, e.event_name,
			e.block_number, e.block_timestamp, e.transaction_hash,
			e.transaction_index, e.log_index, e.args, e.raw_log, e.created_at
		FROM events e
		WHERE 1=1
	`

	// Build WHERE conditions
	whereClause, args := qb.buildEventWhereClause(query)
	queryStr := baseQuery + whereClause

	// Add ORDER BY
	queryStr += " ORDER BY e.block_number DESC, e.log_index ASC"

	// Add LIMIT
	limit := qb.getLimit(query.First, query.Last, 20)
	if limit > 0 {
		queryStr += fmt.Sprintf(" LIMIT %d", limit)
	}

	// Execute query
	rows, err := qb.db.QueryContext(ctx, queryStr, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to execute event query: %w", err)
	}
	defer rows.Close()

	// Parse results
	events, err := qb.parseEvents(rows)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to parse events: %w", err)
	}

	// Get total count (without LIMIT)
	countQuery := `
		SELECT COUNT(*)
		FROM events e
		WHERE 1=1
	` + whereClause

	var totalCount int32
	if err := qb.db.QueryRowContext(ctx, countQuery, args...).Scan(&totalCount); err != nil {
		qb.logger.Warn("Failed to get total count", zap.Error(err))
		totalCount = int32(len(events))
	}

	return events, totalCount, nil
}

// BuildAddressQuery builds and executes a query for events by address
func (qb *QueryBuilder) BuildAddressQuery(ctx context.Context, query *service.AddressQuery) ([]*models.Event, int32, error) {
	// Build the base query with JSONB search
	baseQuery := `
		SELECT 
			e.id, e.contract_id, e.contract_address, e.event_name,
			e.block_number, e.block_timestamp, e.transaction_hash,
			e.transaction_index, e.log_index, e.args, e.raw_log, e.created_at
		FROM events e
		WHERE e.args @> $1
	`

	args := []interface{}{fmt.Sprintf(`{"from": "%s"}`, query.Address)}

	// Add contract filter if specified
	if query.ContractAddress != nil {
		baseQuery += " AND e.contract_address = $2"
		args = append(args, *query.ContractAddress)
	}

	// Add ORDER BY
	baseQuery += " ORDER BY e.block_number DESC, e.log_index ASC"

	// Add LIMIT
	limit := qb.getLimit(query.First, query.Last, 20)
	if limit > 0 {
		baseQuery += fmt.Sprintf(" LIMIT %d", limit)
	}

	// Execute query
	rows, err := qb.db.QueryContext(ctx, baseQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to execute address query: %w", err)
	}
	defer rows.Close()

	// Parse results
	events, err := qb.parseEvents(rows)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to parse events: %w", err)
	}

	// Get total count
	countQuery := `
		SELECT COUNT(*)
		FROM events e
		WHERE e.args @> $1
	`
	if query.ContractAddress != nil {
		countQuery += " AND e.contract_address = $2"
	}

	var totalCount int32
	if err := qb.db.QueryRowContext(ctx, countQuery, args...).Scan(&totalCount); err != nil {
		qb.logger.Warn("Failed to get total count", zap.Error(err))
		totalCount = int32(len(events))
	}

	return events, totalCount, nil
}

// BuildTransactionQuery builds and executes a query for events by transaction
func (qb *QueryBuilder) BuildTransactionQuery(ctx context.Context, query *service.TransactionQuery) ([]*models.Event, error) {
	queryStr := `
		SELECT 
			e.id, e.contract_id, e.contract_address, e.event_name,
			e.block_number, e.block_timestamp, e.transaction_hash,
			e.transaction_index, e.log_index, e.args, e.raw_log, e.created_at
		FROM events e
		WHERE e.transaction_hash = $1
		ORDER BY e.log_index ASC
	`

	rows, err := qb.db.QueryContext(ctx, queryStr, query.TransactionHash)
	if err != nil {
		return nil, fmt.Errorf("failed to execute transaction query: %w", err)
	}
	defer rows.Close()

	events, err := qb.parseEvents(rows)
	if err != nil {
		return nil, fmt.Errorf("failed to parse events: %w", err)
	}

	return events, nil
}

// BuildStatsQuery builds and executes a query for contract statistics
func (qb *QueryBuilder) BuildStatsQuery(ctx context.Context, query *service.StatsQuery) (*service.StatsResponse, error) {
	// Get total events count
	var totalEvents int64
	countQuery := `SELECT COUNT(*) FROM events WHERE contract_address = $1`
	if err := qb.db.QueryRowContext(ctx, countQuery, query.ContractAddress).Scan(&totalEvents); err != nil {
		return nil, fmt.Errorf("failed to get total events: %w", err)
	}

	// Get latest indexed block
	var latestBlock int64
	latestQuery := `SELECT MAX(block_number) FROM events WHERE contract_address = $1`
	if err := qb.db.QueryRowContext(ctx, latestQuery, query.ContractAddress).Scan(&latestBlock); err != nil {
		return nil, fmt.Errorf("failed to get latest block: %w", err)
	}

	// Get current chain block (simplified - in production you'd get this from a chain client)
	currentBlock := latestBlock // This would be fetched from RPC in production

	// Calculate indexer delay (simplified)
	indexerDelay := int64(0) // This would be calculated based on current time vs block timestamp

	// Get last updated time
	var lastUpdated time.Time
	lastUpdatedQuery := `SELECT MAX(created_at) FROM events WHERE contract_address = $1`
	if err := qb.db.QueryRowContext(ctx, lastUpdatedQuery, query.ContractAddress).Scan(&lastUpdated); err != nil {
		lastUpdated = time.Now()
	}

	return &service.StatsResponse{
		ContractAddress: query.ContractAddress,
		TotalEvents:     totalEvents,
		LatestBlock:     latestBlock,
		CurrentBlock:    currentBlock,
		IndexerDelay:    indexerDelay,
		LastUpdated:     lastUpdated,
	}, nil
}

// buildEventWhereClause builds the WHERE clause for event queries
func (qb *QueryBuilder) buildEventWhereClause(query *service.EventQuery) (string, []interface{}) {
	var conditions []string
	var args []interface{}
	argIndex := 1

	if query.ContractAddress != nil {
		conditions = append(conditions, fmt.Sprintf("e.contract_address = $%d", argIndex))
		args = append(args, *query.ContractAddress)
		argIndex++
	}

	if query.EventName != nil {
		conditions = append(conditions, fmt.Sprintf("e.event_name = $%d", argIndex))
		args = append(args, *query.EventName)
		argIndex++
	}

	if query.FromBlock != nil {
		conditions = append(conditions, fmt.Sprintf("e.block_number >= $%d", argIndex))
		args = append(args, *query.FromBlock)
		argIndex++
	}

	if query.ToBlock != nil {
		conditions = append(conditions, fmt.Sprintf("e.block_number <= $%d", argIndex))
		args = append(args, *query.ToBlock)
		argIndex++
	}

	if query.TransactionHash != nil {
		conditions = append(conditions, fmt.Sprintf("e.transaction_hash = $%d", argIndex))
		args = append(args, *query.TransactionHash)
		argIndex++
	}

	// Handle address filtering using JSONB
	if len(query.Addresses) > 0 {
		addressConditions := make([]string, len(query.Addresses))
		for i, addr := range query.Addresses {
			addressConditions[i] = fmt.Sprintf("e.args @> $%d", argIndex)
			args = append(args, fmt.Sprintf(`{"from": "%s"}`, addr))
			argIndex++
		}
		conditions = append(conditions, "("+strings.Join(addressConditions, " OR ")+")")
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = " AND " + strings.Join(conditions, " AND ")
	}

	return whereClause, args
}

// parseEvents parses database rows into Event models
func (qb *QueryBuilder) parseEvents(rows *sql.Rows) ([]*models.Event, error) {
	var events []*models.Event

	for rows.Next() {
		var event models.Event
		var argsJSON string
		var rawLogJSON string

		err := rows.Scan(
			&event.ID,
			&event.ContractID,
			&event.ContractAddress,
			&event.EventName,
			&event.BlockNumber,
			&event.BlockTimestamp,
			&event.TxHash,
			&event.TxIndex,
			&event.LogIndex,
			&argsJSON,
			&rawLogJSON,
			&event.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan event row: %w", err)
		}

		// Parse JSONB args
		if err := event.Args.UnmarshalJSON([]byte(argsJSON)); err != nil {
			qb.logger.Warn("Failed to parse event args", zap.Error(err))
			event.Args = models.JSONB{}
		}

		// Parse raw log if present
		if rawLogJSON != "" {
			if err := event.RawLog.UnmarshalJSON([]byte(rawLogJSON)); err != nil {
				qb.logger.Warn("Failed to parse raw log", zap.Error(err))
			}
		}

		events = append(events, &event)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over rows: %w", err)
	}

	return events, nil
}

// getLimit returns the appropriate limit for pagination
func (qb *QueryBuilder) getLimit(first, last *int32, defaultLimit int32) int32 {
	if first != nil && *first > 0 {
		return *first
	}
	if last != nil && *last > 0 {
		return *last
	}
	return defaultLimit
}
