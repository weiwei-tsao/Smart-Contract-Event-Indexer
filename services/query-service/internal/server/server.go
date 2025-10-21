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
	"github.com/smart-contract-event-indexer/query-service/internal/service"
	"github.com/smart-contract-event-indexer/query-service/internal/config"
	"github.com/smart-contract-event-indexer/shared/utils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

// QueryServiceServer implements the gRPC QueryService
type QueryServiceServer struct {
	db          *sql.DB
	redisClient *redis.Client
	cache       *cache.CacheManager
	queryService *service.QueryService
	logger      utils.Logger
	config      *config.Config

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

	// Register the service
	// Note: We'll need to implement the actual gRPC service interface
	// For now, we'll create a placeholder
	// grpcServer.RegisterService(&proto.QueryService_ServiceDesc, server)

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
