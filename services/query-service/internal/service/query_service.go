package service

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"time"

	"github.com/smart-contract-event-indexer/query-service/internal/cache"
	"github.com/smart-contract-event-indexer/query-service/internal/config"
	"github.com/smart-contract-event-indexer/query-service/internal/optimizer"
	"github.com/smart-contract-event-indexer/shared/models"
	"go.uber.org/zap"
)

// QueryService handles query operations
type QueryService struct {
	db          *sql.DB
	cache       *cache.CacheManager
	logger      *zap.Logger
	config      *config.Config
	queryBuilder *optimizer.QueryBuilder
}

// NewQueryService creates a new QueryService
func NewQueryService(
	db *sql.DB,
	cache *cache.CacheManager,
	logger *zap.Logger,
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

// EventQuery represents a query for events
type EventQuery struct {
	ContractAddress *string
	EventName       *string
	FromBlock       *int64
	ToBlock         *int64
	Addresses       []string
	TransactionHash *string
	First           *int32
	After           *string
	Before          *string
	Last            *int32
}

// EventResponse represents the response for event queries
type EventResponse struct {
	Events    []*models.Event
	TotalCount int32
	PageInfo  *PageInfo
}

// PageInfo represents pagination information
type PageInfo struct {
	HasNextPage     bool
	HasPreviousPage bool
	StartCursor     *string
	EndCursor       *string
}

// AddressQuery represents a query for events by address
type AddressQuery struct {
	Address         string
	ContractAddress *string
	First           *int32
	After           *string
	Before          *string
	Last            *int32
}

// TransactionQuery represents a query for events by transaction
type TransactionQuery struct {
	TransactionHash string
}

// StatsQuery represents a query for contract statistics
type StatsQuery struct {
	ContractAddress string
}

// StatsResponse represents contract statistics
type StatsResponse struct {
	ContractAddress string
	TotalEvents     int64
	LatestBlock     int64
	CurrentBlock    int64
	IndexerDelay    int64
	LastUpdated     time.Time
}

// GetEvents retrieves events based on filter criteria
func (s *QueryService) GetEvents(ctx context.Context, query *EventQuery) (*EventResponse, error) {
	// Generate cache key
	cacheKey, err := s.generateCacheKey("query", query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate cache key: %w", err)
	}

	// Try to get from cache first
	var response *EventResponse
	if err := s.cache.Get(ctx, cacheKey, &response); err == nil {
		s.logger.Debug("Cache hit for events query", zap.String("key", cacheKey.String()))
		return response, nil
	}

	// Build and execute query
	events, totalCount, err := s.queryBuilder.BuildEventQuery(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to build event query: %w", err)
	}

	// Build response
	response = &EventResponse{
		Events:     events,
		TotalCount: totalCount,
		PageInfo:   s.buildPageInfo(events, query),
	}

	// Cache the response
	ttl := s.getCacheTTL(query)
	if err := s.cache.Set(ctx, cacheKey, response, ttl); err != nil {
		s.logger.Warn("Failed to cache events query", zap.Error(err))
	}

	return response, nil
}

// GetEventsByAddress retrieves events involving a specific address
func (s *QueryService) GetEventsByAddress(ctx context.Context, query *AddressQuery) (*EventResponse, error) {
	// Generate cache key
	cacheKey, err := s.generateAddressCacheKey("address", query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate cache key: %w", err)
	}

	// Try to get from cache first
	var response *EventResponse
	if err := s.cache.Get(ctx, cacheKey, &response); err == nil {
		s.logger.Debug("Cache hit for address query", zap.String("key", cacheKey.String()))
		return response, nil
	}

	// Build and execute query
	events, totalCount, err := s.queryBuilder.BuildAddressQuery(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to build address query: %w", err)
	}

	// Build response
	response = &EventResponse{
		Events:     events,
		TotalCount: totalCount,
		PageInfo:   s.buildAddressPageInfo(events, query),
	}

	// Cache the response
	ttl := s.getAddressCacheTTL(query)
	if err := s.cache.Set(ctx, cacheKey, response, ttl); err != nil {
		s.logger.Warn("Failed to cache address query", zap.Error(err))
	}

	return response, nil
}

// GetEventsByTransaction retrieves events for a specific transaction
func (s *QueryService) GetEventsByTransaction(ctx context.Context, query *TransactionQuery) (*EventResponse, error) {
	// Generate cache key
	cacheKey, err := s.generateTransactionCacheKey("tx", query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate cache key: %w", err)
	}

	// Try to get from cache first
	var response *EventResponse
	if err := s.cache.Get(ctx, cacheKey, &response); err == nil {
		s.logger.Debug("Cache hit for transaction query", zap.String("key", cacheKey.String()))
		return response, nil
	}

	// Build and execute query
	events, err := s.queryBuilder.BuildTransactionQuery(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to build transaction query: %w", err)
	}

	// Build response
	response = &EventResponse{
		Events:     events,
		TotalCount: int32(len(events)),
		PageInfo: &PageInfo{
			HasNextPage:     false,
			HasPreviousPage: false,
		},
	}

	// Cache the response (transactions are immutable, so longer TTL)
	if err := s.cache.Set(ctx, cacheKey, response, 1*time.Hour); err != nil {
		s.logger.Warn("Failed to cache transaction query", zap.Error(err))
	}

	return response, nil
}

// GetContractStats retrieves statistics for a contract
func (s *QueryService) GetContractStats(ctx context.Context, query *StatsQuery) (*StatsResponse, error) {
	// Generate cache key
	cacheKey, err := s.generateStatsCacheKey("stats", query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate cache key: %w", err)
	}

	// Try to get from cache first
	var response *StatsResponse
	if err := s.cache.Get(ctx, cacheKey, &response); err == nil {
		s.logger.Debug("Cache hit for stats query", zap.String("key", cacheKey.String()))
		return response, nil
	}

	// Build and execute query
	stats, err := s.queryBuilder.BuildStatsQuery(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to build stats query: %w", err)
	}

	// Cache the response (stats change less frequently)
	if err := s.cache.Set(ctx, cacheKey, stats, 5*time.Minute); err != nil {
		s.logger.Warn("Failed to cache stats query", zap.Error(err))
	}

	return stats, nil
}

// generateCacheKey generates a cache key for event queries
func (s *QueryService) generateCacheKey(cacheType string, query *EventQuery) (*cache.CacheKey, error) {
	hash, err := cache.GenerateHash(query)
	if err != nil {
		return nil, err
	}

	return cache.NewCacheKey(cacheType, hash, "v1"), nil
}

// generateAddressCacheKey generates a cache key for address queries
func (s *QueryService) generateAddressCacheKey(cacheType string, query *AddressQuery) (*cache.CacheKey, error) {
	hash, err := cache.GenerateHash(query)
	if err != nil {
		return nil, err
	}

	return cache.NewCacheKey(cacheType, hash, "v1"), nil
}

// generateTransactionCacheKey generates a cache key for transaction queries
func (s *QueryService) generateTransactionCacheKey(cacheType string, query *TransactionQuery) (*cache.CacheKey, error) {
	hash, err := cache.GenerateHash(query)
	if err != nil {
		return nil, err
	}

	return cache.NewCacheKey(cacheType, hash, "v1"), nil
}

// generateStatsCacheKey generates a cache key for stats queries
func (s *QueryService) generateStatsCacheKey(cacheType string, query *StatsQuery) (*cache.CacheKey, error) {
	hash, err := cache.GenerateHash(query)
	if err != nil {
		return nil, err
	}

	return cache.NewCacheKey(cacheType, hash, "v1"), nil
}

// buildPageInfo builds pagination information for event queries
func (s *QueryService) buildPageInfo(events []*models.Event, query *EventQuery) *PageInfo {
	if len(events) == 0 {
		return &PageInfo{
			HasNextPage:     false,
			HasPreviousPage: false,
		}
	}

	// Simple cursor-based pagination using event ID
	hasNext := false
	hasPrevious := false
	var startCursor, endCursor *string

	if len(events) > 0 {
		firstID := strconv.FormatInt(events[0].ID, 10)
		lastID := strconv.FormatInt(events[len(events)-1].ID, 10)
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

	return &PageInfo{
		HasNextPage:     hasNext,
		HasPreviousPage: hasPrevious,
		StartCursor:     startCursor,
		EndCursor:       endCursor,
	}
}

// buildAddressPageInfo builds pagination information for address queries
func (s *QueryService) buildAddressPageInfo(events []*models.Event, query *AddressQuery) *PageInfo {
	if len(events) == 0 {
		return &PageInfo{
			HasNextPage:     false,
			HasPreviousPage: false,
		}
	}

	// Simple cursor-based pagination using event ID
	hasNext := false
	hasPrevious := false
	var startCursor, endCursor *string

	if len(events) > 0 {
		firstID := strconv.FormatInt(events[0].ID, 10)
		lastID := strconv.FormatInt(events[len(events)-1].ID, 10)
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

	return &PageInfo{
		HasNextPage:     hasNext,
		HasPreviousPage: hasPrevious,
		StartCursor:     startCursor,
		EndCursor:       endCursor,
	}
}

// getCacheTTL returns the appropriate TTL for event queries
func (s *QueryService) getCacheTTL(query *EventQuery) time.Duration {
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
func (s *QueryService) getAddressCacheTTL(query *AddressQuery) time.Duration {
	// Address queries are often for recent activity
	return 1 * time.Minute
}
