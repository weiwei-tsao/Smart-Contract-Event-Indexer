package service

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/smart-contract-event-indexer/query-service/internal/cache"
	"github.com/smart-contract-event-indexer/query-service/internal/config"
	"github.com/smart-contract-event-indexer/query-service/internal/optimizer"
	"github.com/smart-contract-event-indexer/query-service/internal/types"
	"github.com/smart-contract-event-indexer/shared/models"
	"github.com/smart-contract-event-indexer/shared/utils"
)

// QueryService handles query operations
type QueryService struct {
	db          *sql.DB
	cache       *cache.CacheManager
	logger      utils.Logger
	config      *config.Config
	queryBuilder *optimizer.QueryBuilder
}

// NewQueryService creates a new QueryService
func NewQueryService(
	db *sql.DB,
	cache *cache.CacheManager,
	logger utils.Logger,
	cfg *config.Config,
) *QueryService {
	queryBuilder := optimizer.NewQueryBuilder(db, logger)

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
	// Generate cache key
	cacheKey, err := s.generateCacheKey("query", query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate cache key: %w", err)
	}

	// Try to get from cache first
	var response *types.EventResponse
	if err := s.cache.Get(ctx, cacheKey, &response); err == nil {
		s.logger.Debug("Cache hit for events query", "key", cacheKey.String())
		return response, nil
	}

	// Build and execute query
	events, totalCount, err := s.queryBuilder.BuildEventQuery(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to build event query: %w", err)
	}

	// Build response
	response = &types.EventResponse{
		Events:     events,
		TotalCount: totalCount,
		PageInfo:   s.buildPageInfo(events, query),
	}

	// Cache the response
	ttl := s.getCacheTTL(query)
	if err := s.cache.Set(ctx, cacheKey, response, ttl); err != nil {
		s.logger.Warn("Failed to cache events query", "error", err)
	}

	return response, nil
}

// GetEventsByAddress retrieves events involving a specific address
func (s *QueryService) GetEventsByAddress(ctx context.Context, query *types.AddressQuery) (*types.EventResponse, error) {
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

	// Cache the response
	ttl := s.getAddressCacheTTL(query)
	if err := s.cache.Set(ctx, cacheKey, response, ttl); err != nil {
		s.logger.Warn("Failed to cache address query", "error", err)
	}

	return response, nil
}

// GetEventsByTransaction retrieves events for a specific transaction
func (s *QueryService) GetEventsByTransaction(ctx context.Context, query *types.TransactionQuery) (*types.EventResponse, error) {
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
	}

	return response, nil
}

// GetContractStats retrieves statistics for a contract
func (s *QueryService) GetContractStats(ctx context.Context, query *types.StatsQuery) (*types.StatsResponse, error) {
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
	stats, err := s.queryBuilder.BuildStatsQuery(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to build stats query: %w", err)
	}

	// Cache the response (stats change less frequently)
	if err := s.cache.Set(ctx, cacheKey, stats, 5*time.Minute); err != nil {
		s.logger.Warn("Failed to cache stats query", "error", err)
	}

	return stats, nil
}

// generateCacheKey generates a cache key for event queries
func (s *QueryService) generateCacheKey(cacheType string, query *types.EventQuery) (*cache.CacheKey, error) {
	hash, err := cache.GenerateHash(query)
	if err != nil {
		return nil, err
	}

	return cache.NewCacheKey(cacheType, hash, "v1"), nil
}

// generateAddressCacheKey generates a cache key for address queries
func (s *QueryService) generateAddressCacheKey(cacheType string, query *types.AddressQuery) (*cache.CacheKey, error) {
	hash, err := cache.GenerateHash(query)
	if err != nil {
		return nil, err
	}

	return cache.NewCacheKey(cacheType, hash, "v1"), nil
}

// generateTransactionCacheKey generates a cache key for transaction queries
func (s *QueryService) generateTransactionCacheKey(cacheType string, query *types.TransactionQuery) (*cache.CacheKey, error) {
	hash, err := cache.GenerateHash(query)
	if err != nil {
		return nil, err
	}

	return cache.NewCacheKey(cacheType, hash, "v1"), nil
}

// generateStatsCacheKey generates a cache key for stats queries
func (s *QueryService) generateStatsCacheKey(cacheType string, query *types.StatsQuery) (*cache.CacheKey, error) {
	hash, err := cache.GenerateHash(query)
	if err != nil {
		return nil, err
	}

	return cache.NewCacheKey(cacheType, hash, "v1"), nil
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
