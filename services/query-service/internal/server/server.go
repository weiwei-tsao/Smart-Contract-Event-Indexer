package server

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/redis/go-redis/v9"
	"github.com/smart-contract-event-indexer/query-service/internal/cache"
	"github.com/smart-contract-event-indexer/query-service/internal/config"
	"github.com/smart-contract-event-indexer/query-service/internal/service"
	"github.com/smart-contract-event-indexer/query-service/internal/types"
	"github.com/smart-contract-event-indexer/shared/models"
	protoapi "github.com/smart-contract-event-indexer/shared/proto"
	"github.com/smart-contract-event-indexer/shared/utils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// QueryServiceServer implements the gRPC QueryService
type QueryServiceServer struct {
	db           *sql.DB
	redisClient  *redis.Client
	cache        *cache.CacheManager
	queryService *service.QueryService
	logger       utils.Logger
	config       *config.Config

	// Metrics
	queryDuration *prometheus.HistogramVec
	queryTotal    *prometheus.CounterVec
	cacheHits     *prometheus.CounterVec
	cacheMisses   *prometheus.CounterVec
}

// NewQueryServiceServer creates a new QueryServiceServer
func NewQueryServiceServer(
	db *sql.DB,
	redisClient *redis.Client,
	logger utils.Logger,
	cfg *config.Config,
) *grpc.Server {
	// Initialize cache manager
	cacheManager := cache.NewCacheManager(redisClient, logger, cfg.CacheTTL)

	// Initialize query service
	queryService := service.NewQueryService(db, cacheManager, logger, cfg)

	// Create server instance
	server := &QueryServiceServer{
		db:           db,
		redisClient:  redisClient,
		cache:        cacheManager,
		queryService: queryService,
		logger:       logger,
		config:       cfg,
	}

	// Initialize metrics
	server.initMetrics()

	// Create gRPC server with interceptors
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(server.unaryInterceptor),
		grpc.StreamInterceptor(server.streamInterceptor),
	)

	// Register the service implementation
	protoapi.RegisterQueryServiceServer(grpcServer, server)

	return grpcServer
}

// unaryInterceptor provides logging and metrics for unary RPC calls
func (s *QueryServiceServer) unaryInterceptor(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (interface{}, error) {
	start := time.Now()

	// Log the request
	s.logger.Info("gRPC request", "method", info.FullMethod, "request", req)

	// Call the handler
	resp, err := handler(ctx, req)

	// Record metrics
	duration := time.Since(start)
	s.queryDuration.WithLabelValues(info.FullMethod, status.Code(err).String()).Observe(duration.Seconds())
	s.queryTotal.WithLabelValues(info.FullMethod, status.Code(err).String()).Inc()

	// Log the response
	if err != nil {
		s.logger.Error("gRPC request failed", "method", info.FullMethod, "error", err, "duration", duration)
	} else {
		s.logger.Info("gRPC request completed", "method", info.FullMethod, "duration", duration)
	}

	return resp, err
}

// streamInterceptor provides logging and metrics for streaming RPC calls
func (s *QueryServiceServer) streamInterceptor(
	srv interface{},
	ss grpc.ServerStream,
	info *grpc.StreamServerInfo,
	handler grpc.StreamHandler,
) error {
	start := time.Now()

	s.logger.Info("gRPC stream request", "method", info.FullMethod)

	err := handler(srv, ss)

	duration := time.Since(start)
	s.queryDuration.WithLabelValues(info.FullMethod, status.Code(err).String()).Observe(duration.Seconds())
	s.queryTotal.WithLabelValues(info.FullMethod, status.Code(err).String()).Inc()

	if err != nil {
		s.logger.Error("gRPC stream request failed", "method", info.FullMethod, "error", err, "duration", duration)
	} else {
		s.logger.Info("gRPC stream request completed", "method", info.FullMethod, "duration", duration)
	}

	return err
}

// initMetrics initializes Prometheus metrics
func (s *QueryServiceServer) initMetrics() {
	s.queryDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "query_service_request_duration_seconds",
			Help:    "Duration of gRPC requests in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "status"},
	)

	s.queryTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "query_service_requests_total",
			Help: "Total number of gRPC requests",
		},
		[]string{"method", "status"},
	)

	s.cacheHits = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "query_service_cache_hits_total",
			Help: "Total number of cache hits",
		},
		[]string{"cache_type"},
	)

	s.cacheMisses = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "query_service_cache_misses_total",
			Help: "Total number of cache misses",
		},
		[]string{"cache_type"},
	)
}

// Health check method
func (s *QueryServiceServer) HealthCheck(ctx context.Context) error {
	// Check database connection
	if err := s.db.PingContext(ctx); err != nil {
		return fmt.Errorf("database health check failed: %w", err)
	}

	// Check Redis connection
	if err := s.redisClient.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("redis health check failed: %w", err)
	}

	return nil
}

// --- proto interface implementation ---

// GetEvents handles the gRPC call and proxies to the domain query service.
func (s *QueryServiceServer) GetEvents(ctx context.Context, req *protoapi.EventQuery) (*protoapi.EventResponse, error) {
	query := convertEventQuery(req)
	resp, err := s.queryService.GetEvents(ctx, query)
	if err != nil {
		return nil, err
	}
	return convertEventResponse(resp), nil
}

func (s *QueryServiceServer) GetEventsByAddress(ctx context.Context, req *protoapi.AddressQuery) (*protoapi.EventResponse, error) {
	query := convertAddressQuery(req)
	resp, err := s.queryService.GetEventsByAddress(ctx, query)
	if err != nil {
		return nil, err
	}
	return convertEventResponse(resp), nil
}

func (s *QueryServiceServer) GetEventsByTransaction(ctx context.Context, req *protoapi.TransactionQuery) (*protoapi.EventResponse, error) {
	query := &types.TransactionQuery{
		TransactionHash: req.TransactionHash,
	}
	resp, err := s.queryService.GetEventsByTransaction(ctx, query)
	if err != nil {
		return nil, err
	}
	return convertEventResponse(resp), nil
}

func (s *QueryServiceServer) GetContractStats(ctx context.Context, req *protoapi.StatsQuery) (*protoapi.StatsResponse, error) {
	stats, err := s.queryService.GetContractStats(ctx, &types.StatsQuery{
		ContractAddress: req.ContractAddress,
	})
	if err != nil {
		return nil, err
	}
	return convertStatsResponse(stats), nil
}

// --- conversion helpers ---

func convertEventQuery(req *protoapi.EventQuery) *types.EventQuery {
	query := &types.EventQuery{
		Addresses: req.Addresses,
	}

	if req.ContractAddress != "" {
		query.ContractAddress = stringPtr(req.ContractAddress)
	}
	if req.EventName != "" {
		query.EventName = stringPtr(req.EventName)
	}
	if req.FromBlock > 0 {
		query.FromBlock = int64Ptr(req.FromBlock)
	}
	if req.ToBlock > 0 {
		query.ToBlock = int64Ptr(req.ToBlock)
	}
	if req.TransactionHash != "" {
		query.TransactionHash = stringPtr(req.TransactionHash)
	}
	if req.First > 0 {
		query.First = int32Ptr(req.First)
	}
	if req.After != "" {
		query.After = stringPtr(req.After)
	}
	if req.Before != "" {
		query.Before = stringPtr(req.Before)
	}
	if req.Last > 0 {
		query.Last = int32Ptr(req.Last)
	}
	if req.Limit > 0 {
		query.Limit = req.Limit
	}
	if req.Offset > 0 {
		query.Offset = req.Offset
	}

	return query
}

func convertAddressQuery(req *protoapi.AddressQuery) *types.AddressQuery {
	query := &types.AddressQuery{
		Address: req.Address,
	}
	if req.ContractAddress != "" {
		query.ContractAddress = stringPtr(req.ContractAddress)
	}
	if req.First > 0 {
		query.First = int32Ptr(req.First)
	}
	if req.After != "" {
		query.After = stringPtr(req.After)
	}
	if req.Before != "" {
		query.Before = stringPtr(req.Before)
	}
	if req.Last > 0 {
		query.Last = int32Ptr(req.Last)
	}
	if req.Limit > 0 {
		query.Limit = req.Limit
	}
	if req.Offset > 0 {
		query.Offset = req.Offset
	}
	return query
}

func convertEventResponse(resp *types.EventResponse) *protoapi.EventResponse {
	if resp == nil {
		return &protoapi.EventResponse{
			PageInfo: &protoapi.PageInfo{},
		}
	}

	return &protoapi.EventResponse{
		Events:     convertEvents(resp.Events),
		TotalCount: resp.TotalCount,
		PageInfo:   convertPageInfo(resp.PageInfo),
	}
}

func convertEvents(events []*models.Event) []*protoapi.Event {
	if len(events) == 0 {
		return nil
	}

	result := make([]*protoapi.Event, 0, len(events))
	for _, evt := range events {
		if evt == nil {
			continue
		}
		result = append(result, &protoapi.Event{
			Id:               evt.ID,
			ContractAddress:  string(evt.ContractAddress),
			EventName:        evt.EventName,
			BlockNumber:      evt.BlockNumber,
			BlockHash:        string(evt.BlockHash),
			TransactionHash:  string(evt.TransactionHash),
			TransactionIndex: int32(evt.TransactionIndex),
			LogIndex:         int32(evt.LogIndex),
			Args:             convertEventArgs(evt.Args),
			Timestamp:        timestamppb.New(evt.Timestamp),
			CreatedAt:        timestamppb.New(evt.CreatedAt),
		})
	}
	return result
}

func convertEventArgs(args models.JSONB) []*protoapi.EventArg {
	if len(args) == 0 {
		return nil
	}
	result := make([]*protoapi.EventArg, 0, len(args))
	for key, value := range args {
		result = append(result, &protoapi.EventArg{
			Key:   key,
			Value: fmt.Sprintf("%v", value),
			Type:  fmt.Sprintf("%T", value),
		})
	}
	return result
}

func convertPageInfo(info *types.PageInfo) *protoapi.PageInfo {
	if info == nil {
		return &protoapi.PageInfo{}
	}

	page := &protoapi.PageInfo{
		HasNextPage:     info.HasNextPage,
		HasPreviousPage: info.HasPreviousPage,
	}

	if info.StartCursor != nil {
		page.StartCursor = fmt.Sprintf("%d", *info.StartCursor)
	}
	if info.EndCursor != nil {
		page.EndCursor = fmt.Sprintf("%d", *info.EndCursor)
	}

	return page
}

func convertStatsResponse(stats *types.StatsResponse) *protoapi.StatsResponse {
	if stats == nil {
		return &protoapi.StatsResponse{}
	}

	var lastUpdated *timestamppb.Timestamp
	if !stats.LastUpdated.IsZero() {
		lastUpdated = timestamppb.New(stats.LastUpdated)
	}

	return &protoapi.StatsResponse{
		ContractAddress: stats.ContractAddress,
		TotalEvents:     stats.TotalEvents,
		LatestBlock:     stats.LatestBlock,
		CurrentBlock:    stats.CurrentBlock,
		IndexerDelay:    stats.IndexerDelay,
		LastUpdated:     lastUpdated,
	}
}

func stringPtr(v string) *string {
	val := v
	return &val
}

func int32Ptr(v int32) *int32 {
	val := v
	return &val
}

func int64Ptr(v int64) *int64 {
	val := v
	return &val
}
