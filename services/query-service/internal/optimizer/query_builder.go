package optimizer

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/smart-contract-event-indexer/query-service/internal/config"
	"github.com/smart-contract-event-indexer/query-service/internal/types"
	"github.com/smart-contract-event-indexer/shared/models"
	"github.com/smart-contract-event-indexer/shared/utils"
)

// QueryBuilder handles SQL query construction and optimization
type QueryBuilder struct {
	db     *sql.DB
	logger utils.Logger
	config *config.Config
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

// NewQueryBuilder creates a new QueryBuilder
func NewQueryBuilder(db *sql.DB, logger utils.Logger, cfg *config.Config) *QueryBuilder {
	return &QueryBuilder{
		db:     db,
		logger: logger,
		config: cfg,
	}
}

// BuildSimpleEventQuery uses a streamlined SQL path for common filters.
func (qb *QueryBuilder) BuildSimpleEventQuery(ctx context.Context, query *types.EventQuery) ([]*models.Event, int32, error) {
	if query.ContractAddress == nil {
		return nil, 0, fmt.Errorf("simple event query requires contract address")
	}

	ctx, cancel := qb.withTimeout(ctx)
	defer cancel()

	baseQuery := `
		SELECT 
			e.id, e.contract_address, e.event_name,
			e.block_number, e.block_hash, e.transaction_hash,
			e.transaction_index, e.log_index, e.args, e.timestamp, e.created_at
		FROM events e
		WHERE e.contract_address = $1
	`

	args := []interface{}{*query.ContractAddress}
	argIndex := 2

	if query.EventName != nil {
		baseQuery += fmt.Sprintf(" AND e.event_name = $%d", argIndex)
		args = append(args, *query.EventName)
		argIndex++
	}

	if query.FromBlock != nil {
		baseQuery += fmt.Sprintf(" AND e.block_number >= $%d", argIndex)
		args = append(args, *query.FromBlock)
		argIndex++
	}

	if query.ToBlock != nil {
		baseQuery += fmt.Sprintf(" AND e.block_number <= $%d", argIndex)
		args = append(args, *query.ToBlock)
		argIndex++
	}

	order := " ORDER BY e.block_number DESC, e.log_index ASC"
	limitClause, limitArgs := qb.buildLimitClause(query.First, query.Last, query.Limit, query.Offset, argIndex)
	argIndex += len(limitArgs)

	queryStr := baseQuery + order + limitClause
	args = append(args, limitArgs...)

	rows, err := qb.executeRows(ctx, "events.simple", queryStr, args)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	events, err := qb.parseEvents(rows)
	if err != nil {
		return nil, 0, err
	}

	countQuery := "SELECT COUNT(*) FROM events e WHERE e.contract_address = $1"
	countArgs := []interface{}{*query.ContractAddress}
	if query.EventName != nil {
		countQuery += " AND e.event_name = $2"
		countArgs = append(countArgs, *query.EventName)
	}

	var totalCount int32
	if err := qb.queryRow(ctx, "events.simple.count", countQuery, countArgs, &totalCount); err != nil {
		return events, int32(len(events)), nil
	}

	return events, totalCount, nil
}

// BuildEventQuery builds and executes a query for events
func (qb *QueryBuilder) BuildEventQuery(ctx context.Context, query *types.EventQuery) ([]*models.Event, int32, error) {
	ctx, cancel := qb.withTimeout(ctx)
	defer cancel()

	// Build the base query
	baseQuery := `
		SELECT 
			e.id, 	e.contract_address, e.event_name,
			e.block_number, e.block_hash, e.transaction_hash,
			e.transaction_index, e.log_index, e.args, e.timestamp, e.created_at
		FROM events e
		WHERE 1=1
	`

	whereClause, args := qb.buildEventWhereClause(query)
	countArgs := append([]interface{}{}, args...)

	queryStr := baseQuery + whereClause + " ORDER BY e.block_number DESC, e.log_index ASC"
	limitClause, limitArgs := qb.buildLimitClause(query.First, query.Last, query.Limit, query.Offset, len(args)+1)
	queryStr += limitClause
	args = append(args, limitArgs...)

	rows, err := qb.executeRows(ctx, "events.complex", queryStr, args)
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
	if err := qb.queryRow(ctx, "events.complex.count", countQuery, countArgs, &totalCount); err != nil {
		return events, int32(len(events)), nil
	}

	return events, totalCount, nil
}

// BuildAddressQuery builds and executes a query for events by address
func (qb *QueryBuilder) BuildAddressQuery(ctx context.Context, query *types.AddressQuery) ([]*models.Event, int32, error) {
	ctx, cancel := qb.withTimeout(ctx)
	defer cancel()

	baseQuery := `
		SELECT 
			e.id, 	e.contract_address, e.event_name,
			e.block_number, e.block_hash, e.transaction_hash,
			e.transaction_index, e.log_index, e.args, e.timestamp, e.created_at
		FROM events e
		WHERE %s
	`

	filter, filterArgs := qb.buildAddressClause(query.Address, 1)
	args := filterArgs
	argIndex := len(args) + 1

	if query.ContractAddress != nil {
		filter += fmt.Sprintf(" AND e.contract_address = $%d", argIndex)
		args = append(args, *query.ContractAddress)
		argIndex++
	}

	if query.EventName != nil {
		filter += fmt.Sprintf(" AND e.event_name = $%d", argIndex)
		args = append(args, *query.EventName)
		argIndex++
	}

	if query.FromBlock != nil {
		filter += fmt.Sprintf(" AND e.block_number >= $%d", argIndex)
		args = append(args, *query.FromBlock)
		argIndex++
	}

	if query.ToBlock != nil {
		filter += fmt.Sprintf(" AND e.block_number <= $%d", argIndex)
		args = append(args, *query.ToBlock)
		argIndex++
	}

	countArgs := append([]interface{}{}, args...)
	queryStr := fmt.Sprintf(baseQuery, filter) + " ORDER BY e.block_number DESC, e.log_index ASC"
	limitClause, limitArgs := qb.buildLimitClause(query.First, query.Last, query.Limit, query.Offset, argIndex)
	queryStr += limitClause
	args = append(args, limitArgs...)

	rows, err := qb.executeRows(ctx, "events.address", queryStr, args)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to execute address query: %w", err)
	}
	defer rows.Close()

	events, err := qb.parseEvents(rows)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to parse events: %w", err)
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM events e WHERE %s", filter)
	var totalCount int32
	if err := qb.queryRow(ctx, "events.address.count", countQuery, countArgs, &totalCount); err != nil {
		return events, int32(len(events)), nil
	}

	return events, totalCount, nil
}

// BuildTransactionQuery builds and executes a query for events by transaction
func (qb *QueryBuilder) BuildTransactionQuery(ctx context.Context, query *types.TransactionQuery) ([]*models.Event, error) {
	ctx, cancel := qb.withTimeout(ctx)
	defer cancel()

	queryStr := `
		SELECT 
			e.id, 	e.contract_address, e.event_name,
			e.block_number, e.block_hash, e.transaction_hash,
			e.transaction_index, e.log_index, e.args, e.timestamp, e.created_at
		FROM events e
		WHERE e.transaction_hash = $1
		ORDER BY e.log_index ASC
	`

	rows, err := qb.executeRows(ctx, "events.tx", queryStr, []interface{}{query.TransactionHash})
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
func (qb *QueryBuilder) BuildStatsQuery(ctx context.Context, query *types.StatsQuery) (*types.StatsResponse, error) {
	ctx, cancel := qb.withTimeout(ctx)
	defer cancel()

	stats := &types.StatsResponse{
		ContractAddress: query.ContractAddress,
	}

	countQuery := `SELECT COUNT(*) FROM events WHERE contract_address = $1`
	if err := qb.queryRow(ctx, "stats.count", countQuery, []interface{}{query.ContractAddress}, &stats.TotalEvents); err != nil {
		return nil, fmt.Errorf("failed to get total events: %w", err)
	}

	latestQuery := `SELECT COALESCE(MAX(block_number), 0) FROM events WHERE contract_address = $1`
	if err := qb.queryRow(ctx, "stats.latest", latestQuery, []interface{}{query.ContractAddress}, &stats.LatestBlock); err != nil {
		return nil, fmt.Errorf("failed to get latest block: %w", err)
	}

	var currentBlock sql.NullInt64
	var lastUpdated sql.NullTime
	stateQuery := `SELECT last_indexed_block, updated_at FROM indexer_state WHERE contract_address = $1`
	switch err := qb.db.QueryRowContext(ctx, stateQuery, query.ContractAddress).Scan(&currentBlock, &lastUpdated); err {
	case nil:
		stats.CurrentBlock = currentBlock.Int64
		if lastUpdated.Valid {
			stats.LastUpdated = lastUpdated.Time
		}
	case sql.ErrNoRows:
		stats.CurrentBlock = stats.LatestBlock
	default:
		return nil, fmt.Errorf("failed to load indexer state: %w", err)
	}

	if stats.CurrentBlock == 0 {
		stats.CurrentBlock = stats.LatestBlock
	}

	if stats.LastUpdated.IsZero() {
		if err := qb.db.QueryRowContext(ctx, `SELECT COALESCE(MAX(created_at), NOW()) FROM events WHERE contract_address = $1`, query.ContractAddress).Scan(&stats.LastUpdated); err != nil {
			stats.LastUpdated = time.Now()
		}
	}

	if stats.CurrentBlock >= stats.LatestBlock {
		stats.IndexerDelay = stats.CurrentBlock - stats.LatestBlock
	} else {
		stats.IndexerDelay = stats.LatestBlock - stats.CurrentBlock
	}

	var uniqueAddresses sql.NullInt64
	uniqueQuery := `
		WITH addresses AS (
			SELECT LOWER(kv.value) AS addr
			FROM events e
			CROSS JOIN LATERAL jsonb_each_text(e.args) kv(key, value)
			WHERE e.contract_address = $1
			  AND kv.value ~ '^0x[0-9a-fA-F]{40}$'
		)
		SELECT COUNT(DISTINCT addr) FROM addresses
	`
	if err := qb.db.QueryRowContext(ctx, uniqueQuery, query.ContractAddress).Scan(&uniqueAddresses); err == nil && uniqueAddresses.Valid {
		count := uniqueAddresses.Int64
		stats.UniqueAddresses = &count
	}

	return stats, nil
}

// BuildTimeRangeAggregation returns bucketed totals for a contract.
func (qb *QueryBuilder) BuildTimeRangeAggregation(ctx context.Context, query *types.TimeRangeQuery) ([]*types.TimeBucketStat, error) {
	ctx, cancel := qb.withTimeout(ctx)
	defer cancel()

	interval := strings.ToLower(query.Interval)
	if interval == "" {
		interval = "hour"
	}

	intervalDuration := fmt.Sprintf("1 %s", interval)

	sqlQuery := `
		SELECT 
			date_trunc($3, e.timestamp) AS bucket_start,
			date_trunc($3, e.timestamp) + $4::interval AS bucket_end,
			COUNT(*) as total
		FROM events e
		WHERE e.contract_address = $1
		  AND e.timestamp BETWEEN $2 AND $5
	`

	args := []interface{}{
		query.ContractAddress,
		query.From,
		interval,
		intervalDuration,
		query.To,
	}
	if query.EventName != nil {
		sqlQuery += " AND e.event_name = $6"
		args = append(args, *query.EventName)
	}

	sqlQuery += " GROUP BY bucket_start, bucket_end ORDER BY bucket_start ASC"

	rows, err := qb.executeRows(ctx, "aggregate.range", sqlQuery, args)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var buckets []*types.TimeBucketStat
	for rows.Next() {
		var bucket types.TimeBucketStat
		if err := rows.Scan(&bucket.BucketStart, &bucket.BucketEnd, &bucket.EventCount); err != nil {
			return nil, err
		}
		buckets = append(buckets, &bucket)
	}

	return buckets, rows.Err()
}

// BuildTopAddresses ranks addresses by activity.
func (qb *QueryBuilder) BuildTopAddresses(ctx context.Context, query *types.TopNQuery) ([]*types.TopAddressStat, error) {
	ctx, cancel := qb.withTimeout(ctx)
	defer cancel()

	window := query.Window
	if window <= 0 {
		window = 24 * time.Hour
	}

	limit := query.Limit
	if limit <= 0 {
		limit = 10
	}

	topQuery := `
		SELECT LOWER(kv.value) AS address, COUNT(*) AS total
		FROM events e
		CROSS JOIN LATERAL jsonb_each_text(e.args) kv(key, value)
		WHERE e.contract_address = $1
		  AND kv.value ~ '^0x[0-9a-fA-F]{40}$'
		  AND e.timestamp >= $2
	`

	args := []interface{}{
		query.ContractAddress,
		time.Now().Add(-window),
	}

	argIndex := 3
	if query.EventName != nil {
		topQuery += fmt.Sprintf(" AND e.event_name = $%d", argIndex)
		args = append(args, *query.EventName)
		argIndex++
	}

	topQuery += fmt.Sprintf(" GROUP BY address ORDER BY total DESC LIMIT $%d", argIndex)
	args = append(args, limit)

	rows, err := qb.executeRows(ctx, "aggregate.top", topQuery, args)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*types.TopAddressStat
	for rows.Next() {
		var stat types.TopAddressStat
		if err := rows.Scan(&stat.Address, &stat.EventCount); err != nil {
			return nil, err
		}
		results = append(results, &stat)
	}

	return results, rows.Err()
}

// buildEventWhereClause builds the WHERE clause for event queries
func (qb *QueryBuilder) buildEventWhereClause(query *types.EventQuery) (string, []interface{}) {
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

		err := rows.Scan(
			&event.ID,
			&event.ContractAddress,
			&event.EventName,
			&event.BlockNumber,
			&event.BlockHash,
			&event.TransactionHash,
			&event.TransactionIndex,
			&event.LogIndex,
			&argsJSON,
			&event.Timestamp,
			&event.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan event row: %w", err)
		}

		// Parse JSONB args
		if err := json.Unmarshal([]byte(argsJSON), &event.Args); err != nil {
			qb.logger.Warn("Failed to parse event args", "error", err)
			event.Args = models.JSONB{}
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

func (qb *QueryBuilder) buildLimitClause(first, last *int32, limit, offset int32, startIndex int) (string, []interface{}) {
	maxLimit := int32(qb.config.DefaultLimit)
	if maxLimit == 0 {
		maxLimit = 25
	}

	calculated := qb.getLimit(first, last, maxLimit)
	if limit > 0 && (calculated == 0 || limit < calculated) {
		calculated = limit
	}
	if qb.config.MaxQueryLimit > 0 && calculated > int32(qb.config.MaxQueryLimit) {
		calculated = int32(qb.config.MaxQueryLimit)
	}
	if calculated <= 0 {
		calculated = maxLimit
	}

	clause := fmt.Sprintf(" LIMIT $%d", startIndex)
	args := []interface{}{calculated}

	if offset > 0 {
		clause += fmt.Sprintf(" OFFSET $%d", startIndex+1)
		args = append(args, offset)
	}

	return clause, args
}

func (qb *QueryBuilder) executeRows(ctx context.Context, label string, queryStr string, args []interface{}) (*sql.Rows, error) {
	start := time.Now()
	rows, err := qb.db.QueryContext(ctx, queryStr, args...)
	qb.observeQuery(label, queryStr, args, start, err)
	if err != nil {
		return nil, err
	}
	return rows, nil
}

func (qb *QueryBuilder) queryRow(ctx context.Context, label string, queryStr string, args []interface{}, dest interface{}) error {
	start := time.Now()
	err := qb.db.QueryRowContext(ctx, queryStr, args...).Scan(dest)
	qb.observeQuery(label, queryStr, args, start, err)
	return err
}

func (qb *QueryBuilder) buildAddressClause(address string, startIndex int) (string, []interface{}) {
	fields := []string{"from", "to", "owner", "spender"}
	var conditions []string
	args := make([]interface{}, 0, len(fields))
	idx := startIndex

	for _, field := range fields {
		conditions = append(conditions, fmt.Sprintf("e.args @> $%d", idx))
		args = append(args, fmt.Sprintf(`{"%s": "%s"}`, field, address))
		idx++
	}

	// Fallback generic contains check using ilike on JSON text
	conditions = append(conditions, fmt.Sprintf("encode(e.args::bytea, 'escape') ILIKE $%d", idx))
	args = append(args, fmt.Sprintf("%%%s%%", strings.TrimPrefix(strings.ToLower(address), "0x")))

	return "(" + strings.Join(conditions, " OR ") + ")", args
}

func (qb *QueryBuilder) observeQuery(label, queryStr string, args []interface{}, start time.Time, err error) {
	duration := time.Since(start)
	if err != nil && err != sql.ErrNoRows {
		qb.logger.Error("Query execution failed", "label", label, "error", err, "duration", duration)
		return
	}

	if qb.config == nil || qb.config.SlowQueryThreshold <= 0 || duration <= qb.config.SlowQueryThreshold {
		return
	}

	qb.logSlowQuery(label, queryStr, args, duration)
}

func (qb *QueryBuilder) withTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if qb.config == nil || qb.config.QueryTimeout <= 0 {
		return context.WithCancel(ctx)
	}
	return context.WithTimeout(ctx, qb.config.QueryTimeout)
}

func (qb *QueryBuilder) logSlowQuery(label, queryStr string, args []interface{}, duration time.Duration) {
	qb.logger.Warn("Slow SQL detected", "label", label, "duration", duration)
	if !qb.shouldExplain() {
		return
	}

	if plan, err := qb.captureExplain(queryStr, args); err == nil {
		qb.logger.Info("Query plan", "label", label, "plan", plan)
	} else {
		qb.logger.Warn("Failed to capture query plan", "label", label, "error", err)
	}
}

func (qb *QueryBuilder) captureExplain(queryStr string, args []interface{}) (string, error) {
	if qb.config == nil {
		return "", fmt.Errorf("config not initialized")
	}

	ctx, cancel := context.WithTimeout(context.Background(), qb.config.QueryTimeout)
	defer cancel()

	rows, err := qb.db.QueryContext(ctx, "EXPLAIN (FORMAT JSON) "+queryStr, args...)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	var plan string
	for rows.Next() {
		if err := rows.Scan(&plan); err != nil {
			return "", err
		}
	}

	return plan, rows.Err()
}

func (qb *QueryBuilder) shouldExplain() bool {
	if qb.config == nil || qb.config.ExplainPlanSample <= 0 {
		return false
	}
	return rand.Intn(qb.config.ExplainPlanSample) == 0
}
