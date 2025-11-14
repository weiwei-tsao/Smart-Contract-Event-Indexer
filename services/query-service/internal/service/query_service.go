package service

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/smart-contract-event-indexer/query-service/internal/cache"
	"github.com/smart-contract-event-indexer/query-service/internal/config"
	"github.com/smart-contract-event-indexer/query-service/internal/optimizer"
	"github.com/smart-contract-event-indexer/query-service/internal/types"
	"github.com/smart-contract-event-indexer/shared/models"
	"github.com/smart-contract-event-indexer/shared/utils"
)

const cacheVersion = "v2"

type queryPath string

const (
	queryPathSimple  queryPath = "simple"
	queryPathComplex queryPath = "complex"
)

// QueryService handles query operations
type QueryService struct {
	db           *sql.DB
	cache        *cache.CacheManager
	logger       utils.Logger
	config       *config.Config
	queryBuilder *optimizer.QueryBuilder
}

// NewQueryService creates a new QueryService
func NewQueryService(
	db *sql.DB,
	cache *cache.CacheManager,
	logger utils.Logger,
	cfg *config.Config,
) *QueryService {
	queryBuilder := optimizer.NewQueryBuilder(db, logger, cfg)

	return &QueryService{
		db:           db,
		cache:        cache,
		logger:       logger,
		config:       cfg,
		queryBuilder: queryBuilder,
	}
}

// GetEvents retrieves events based on filter criteria
func (s *QueryService) GetEvents(ctx context.Context, query *types.EventQuery) (*types.EventResponse, error) {
	ctx, cancel := s.withQueryTimeout(ctx)
	defer cancel()

	cacheKey, err := s.generateCacheKey("query", query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate cache key: %w", err)
	}

	var response *types.EventResponse
	if err := s.cache.Get(ctx, cacheKey, &response); err == nil {
		s.logger.Debug("Cache hit for events query", "key", cacheKey.String())
		return response, nil
	}

	if s.cache.IsNegative(ctx, cacheKey) {
		s.logger.Debug("Negative cache bypass for events query", "key", cacheKey.String())
		return s.emptyEventResponse(query), nil
	}

	queryPath := s.determineEventQueryPath(query)
	start := time.Now()

	var (
		events     []*models.Event
		totalCount int32
	)

	switch queryPath {
	case queryPathSimple:
		events, totalCount, err = s.queryBuilder.BuildSimpleEventQuery(ctx, query)
	default:
		events, totalCount, err = s.queryBuilder.BuildEventQuery(ctx, query)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to build event query: %w", err)
	}

	response = &types.EventResponse{
		Events:     events,
		TotalCount: totalCount,
		PageInfo:   s.buildPageInfo(events, query),
	}

	if len(events) == 0 {
		s.cache.MarkNegative(ctx, cacheKey)
	}

	ttl := s.getCacheTTL(query)
	if err := s.cache.Set(ctx, cacheKey, response, ttl); err != nil {
		s.logger.Warn("Failed to cache events query", "error", err)
	}

	s.logQueryStats("events", queryPath, time.Since(start), len(events))

	return response, nil
}

// GetEventsByAddress retrieves events involving a specific address
func (s *QueryService) GetEventsByAddress(ctx context.Context, query *types.AddressQuery) (*types.EventResponse, error) {
	ctx, cancel := s.withQueryTimeout(ctx)
	defer cancel()

	// Generate cache key
	cacheKey, err := s.generateAddressCacheKey("address", query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate cache key: %w", err)
	}

	// Try to get from cache first
	var response *types.EventResponse
	if err := s.cache.Get(ctx, cacheKey, &response); err == nil {
		s.logger.Debug("Cache hit for address query", "key", cacheKey.String())
		return response, nil
	}

	if s.cache.IsNegative(ctx, cacheKey) {
		return s.emptyAddressResponse(query), nil
	}

	start := time.Now()

	// Build and execute query
	events, totalCount, err := s.queryBuilder.BuildAddressQuery(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to build address query: %w", err)
	}

	// Build response
	response = &types.EventResponse{
		Events:     events,
		TotalCount: totalCount,
		PageInfo:   s.buildAddressPageInfo(events, query),
	}

	if len(events) == 0 {
		s.cache.MarkNegative(ctx, cacheKey)
	}

	// Cache the response
	ttl := s.getAddressCacheTTL(query)
	if err := s.cache.Set(ctx, cacheKey, response, ttl); err != nil {
		s.logger.Warn("Failed to cache address query", "error", err)
	}

	s.logQueryStats("eventsByAddress", queryPathComplex, time.Since(start), len(events))

	return response, nil
}

// GetEventsByTransaction retrieves events for a specific transaction
func (s *QueryService) GetEventsByTransaction(ctx context.Context, query *types.TransactionQuery) (*types.EventResponse, error) {
	ctx, cancel := s.withQueryTimeout(ctx)
	defer cancel()

	// Generate cache key
	cacheKey, err := s.generateTransactionCacheKey("tx", query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate cache key: %w", err)
	}

	// Try to get from cache first
	var response *types.EventResponse
	if err := s.cache.Get(ctx, cacheKey, &response); err == nil {
		s.logger.Debug("Cache hit for transaction query", "key", cacheKey.String())
		return response, nil
	}

	if s.cache.IsNegative(ctx, cacheKey) {
		return &types.EventResponse{
			Events:     []*models.Event{},
			TotalCount: 0,
			PageInfo: &types.PageInfo{
				HasNextPage:     false,
				HasPreviousPage: false,
			},
		}, nil
	}

	start := time.Now()

	// Build and execute query
	events, err := s.queryBuilder.BuildTransactionQuery(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to build transaction query: %w", err)
	}

	// Build response
	response = &types.EventResponse{
		Events:     events,
		TotalCount: int32(len(events)),
		PageInfo: &types.PageInfo{
			HasNextPage:     false,
			HasPreviousPage: false,
		},
	}

	// Cache the response (transactions are immutable, so longer TTL)
	if err := s.cache.Set(ctx, cacheKey, response, 1*time.Hour); err != nil {
		s.logger.Warn("Failed to cache transaction query", "error", err)
	} else if len(events) == 0 {
		s.cache.MarkNegative(ctx, cacheKey)
	}

	s.logQueryStats("eventsByTransaction", queryPathSimple, time.Since(start), len(events))

	return response, nil
}

// GetContractStats retrieves statistics for a contract
func (s *QueryService) GetContractStats(ctx context.Context, query *types.StatsQuery) (*types.StatsResponse, error) {
	ctx, cancel := s.withQueryTimeout(ctx)
	defer cancel()

	// Generate cache key
	cacheKey, err := s.generateStatsCacheKey("stats", query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate cache key: %w", err)
	}

	// Try to get from cache first
	var response *types.StatsResponse
	if err := s.cache.Get(ctx, cacheKey, &response); err == nil {
		s.logger.Debug("Cache hit for stats query", "key", cacheKey.String())
		return response, nil
	}

	// Build and execute query
	start := time.Now()

	stats, err := s.queryBuilder.BuildStatsQuery(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to build stats query: %w", err)
	}

	// Cache the response (stats change less frequently)
	ttl := s.aggregationTTL()
	if err := s.cache.Set(ctx, cacheKey, stats, ttl); err != nil {
		s.logger.Warn("Failed to cache stats query", "error", err)
	}

	s.logQueryStats("contractStats", queryPathSimple, time.Since(start), int(stats.TotalEvents))

	return stats, nil
}

// GetTimeRangeStats returns aggregated event counts bucketed by interval.
func (s *QueryService) GetTimeRangeStats(ctx context.Context, query *types.TimeRangeQuery) ([]*types.TimeBucketStat, error) {
	ctx, cancel := s.withQueryTimeout(ctx)
	defer cancel()

	if err := s.validateInterval(query.Interval); err != nil {
		return nil, err
	}

	cacheKey, err := s.generateAggregationCacheKey("agg:range", query)
	if err != nil {
		return nil, err
	}

	var cached []*types.TimeBucketStat
	if err := s.cache.Get(ctx, cacheKey, &cached); err == nil {
		return cached, nil
	}

	buckets, err := s.queryBuilder.BuildTimeRangeAggregation(ctx, query)
	if err != nil {
		return nil, err
	}

	if err := s.cache.Set(ctx, cacheKey, buckets, s.aggregationTTL()); err != nil {
		s.logger.Warn("Failed to cache range aggregation", "error", err)
	}

	return buckets, nil
}

// GetTopAddresses returns addresses ranked by activity within a window.
func (s *QueryService) GetTopAddresses(ctx context.Context, query *types.TopNQuery) ([]*types.TopAddressStat, error) {
	ctx, cancel := s.withQueryTimeout(ctx)
	defer cancel()

	cacheKey, err := s.generateAggregationCacheKey("agg:top", query)
	if err != nil {
		return nil, err
	}

	var cached []*types.TopAddressStat
	if err := s.cache.Get(ctx, cacheKey, &cached); err == nil {
		return cached, nil
	}

	top, err := s.queryBuilder.BuildTopAddresses(ctx, query)
	if err != nil {
		return nil, err
	}

	if err := s.cache.Set(ctx, cacheKey, top, s.aggregationTTL()); err != nil {
		s.logger.Warn("Failed to cache top addresses", "error", err)
	}

	return top, nil
}

// generateCacheKey generates a cache key for event queries
func (s *QueryService) generateCacheKey(cacheType string, query *types.EventQuery) (*cache.CacheKey, error) {
	hash, err := cache.GenerateHash(query)
	if err != nil {
		return nil, err
	}

	return cache.NewCacheKey(cacheType, hash, cacheVersion), nil
}

// generateAddressCacheKey generates a cache key for address queries
func (s *QueryService) generateAddressCacheKey(cacheType string, query *types.AddressQuery) (*cache.CacheKey, error) {
	hash, err := cache.GenerateHash(query)
	if err != nil {
		return nil, err
	}

	return cache.NewCacheKey(cacheType, hash, cacheVersion), nil
}

// generateTransactionCacheKey generates a cache key for transaction queries
func (s *QueryService) generateTransactionCacheKey(cacheType string, query *types.TransactionQuery) (*cache.CacheKey, error) {
	hash, err := cache.GenerateHash(query)
	if err != nil {
		return nil, err
	}

	return cache.NewCacheKey(cacheType, hash, cacheVersion), nil
}

// generateStatsCacheKey generates a cache key for stats queries
func (s *QueryService) generateStatsCacheKey(cacheType string, query *types.StatsQuery) (*cache.CacheKey, error) {
	hash, err := cache.GenerateHash(query)
	if err != nil {
		return nil, err
	}

	return cache.NewCacheKey(cacheType, hash, cacheVersion), nil
}

func (s *QueryService) generateAggregationCacheKey(cacheType string, query interface{}) (*cache.CacheKey, error) {
	hash, err := cache.GenerateHash(query)
	if err != nil {
		return nil, err
	}
	return cache.NewCacheKey(cacheType, hash, cacheVersion), nil
}

// buildPageInfo builds pagination information for event queries
func (s *QueryService) buildPageInfo(events []*models.Event, query *types.EventQuery) *types.PageInfo {
	if len(events) == 0 {
		return &types.PageInfo{
			HasNextPage:     false,
			HasPreviousPage: false,
		}
	}

	// Simple cursor-based pagination using event ID
	hasNext := false
	hasPrevious := false
	var startCursor, endCursor *int64

	if len(events) > 0 {
		firstID := events[0].ID
		lastID := events[len(events)-1].ID
		startCursor = &firstID
		endCursor = &lastID

		// Check if there are more results (simplified logic)
		if query.First != nil && len(events) == int(*query.First) {
			hasNext = true
		}
		if query.Before != nil {
			hasPrevious = true
		}
	}

	return &types.PageInfo{
		HasNextPage:     hasNext,
		HasPreviousPage: hasPrevious,
		StartCursor:     startCursor,
		EndCursor:       endCursor,
	}
}

// buildAddressPageInfo builds pagination information for address queries
func (s *QueryService) buildAddressPageInfo(events []*models.Event, query *types.AddressQuery) *types.PageInfo {
	if len(events) == 0 {
		return &types.PageInfo{
			HasNextPage:     false,
			HasPreviousPage: false,
		}
	}

	// Simple cursor-based pagination using event ID
	hasNext := false
	hasPrevious := false
	var startCursor, endCursor *int64

	if len(events) > 0 {
		firstID := events[0].ID
		lastID := events[len(events)-1].ID
		startCursor = &firstID
		endCursor = &lastID

		// Check if there are more results (simplified logic)
		if query.First != nil && len(events) == int(*query.First) {
			hasNext = true
		}
		if query.Before != nil {
			hasPrevious = true
		}
	}

	return &types.PageInfo{
		HasNextPage:     hasNext,
		HasPreviousPage: hasPrevious,
		StartCursor:     startCursor,
		EndCursor:       endCursor,
	}
}

// getCacheTTL returns the appropriate TTL for event queries
func (s *QueryService) aggregationTTL() time.Duration {
	if s.config.AggregationCacheTTL > 0 {
		return s.config.AggregationCacheTTL
	}
	return 5 * time.Minute
}

func (s *QueryService) getCacheTTL(query *types.EventQuery) time.Duration {
	// Recent events (last 1000 blocks) have shorter TTL
	if query.ToBlock != nil && query.FromBlock != nil {
		blockRange := *query.ToBlock - *query.FromBlock
		if blockRange < 1000 {
			return 30 * time.Second
		}
	}
	return 5 * time.Minute
}

// getAddressCacheTTL returns the appropriate TTL for address queries
func (s *QueryService) getAddressCacheTTL(query *types.AddressQuery) time.Duration {
	// Address queries are often for recent activity
	return 1 * time.Minute
}

func (s *QueryService) withQueryTimeout(parent context.Context) (context.Context, context.CancelFunc) {
	if s.config.QueryTimeout <= 0 {
		return context.WithCancel(parent)
	}
	return context.WithTimeout(parent, s.config.QueryTimeout)
}

func (s *QueryService) determineEventQueryPath(query *types.EventQuery) queryPath {
	// Complex path if addresses, transaction hashes, or cursor pagination are in play.
	if len(query.Addresses) > 0 ||
		query.TransactionHash != nil ||
		query.After != nil || query.Before != nil {
		return queryPathComplex
	}

	// Simple fast-path if only contract/event filters are applied.
	if query.ContractAddress != nil &&
		(query.EventName != nil || (query.FromBlock == nil && query.ToBlock == nil)) {
		return queryPathSimple
	}

	return queryPathComplex
}

func (s *QueryService) emptyEventResponse(query *types.EventQuery) *types.EventResponse {
	return &types.EventResponse{
		Events:     []*models.Event{},
		TotalCount: 0,
		PageInfo:   s.buildPageInfo(nil, query),
	}
}

func (s *QueryService) emptyAddressResponse(query *types.AddressQuery) *types.EventResponse {
	return &types.EventResponse{
		Events:     []*models.Event{},
		TotalCount: 0,
		PageInfo:   s.buildAddressPageInfo(nil, query),
	}
}

func (s *QueryService) logQueryStats(label string, path queryPath, duration time.Duration, count int) {
	if duration > s.config.SlowQueryThreshold && s.config.SlowQueryThreshold > 0 {
		s.logger.Warn("Slow query detected",
			"label", label,
			"path", path,
			"duration", duration,
			"hits", count,
		)
	}
}

func (s *QueryService) validateInterval(interval string) error {
	switch strings.ToLower(interval) {
	case "minute", "hour", "day":
		return nil
	default:
		return fmt.Errorf("unsupported interval: %s", interval)
	}
}
