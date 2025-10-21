package server

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/redis/go-redis/v9"
	"github.com/smart-contract-event-indexer/admin-service/internal/config"
	"github.com/smart-contract-event-indexer/admin-service/internal/service"
	"github.com/smart-contract-event-indexer/shared/utils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

// AdminServiceServer implements the gRPC AdminService
type AdminServiceServer struct {
	db          *sql.DB
	redisClient *redis.Client
	adminService *service.AdminService
	logger      utils.Logger
	config      *config.Config

	// Metrics
	requestDuration *prometheus.HistogramVec
	requestTotal    *prometheus.CounterVec
}

// NewAdminServiceServer creates a new AdminServiceServer
func NewAdminServiceServer(
	db *sql.DB,
	redisClient *redis.Client,
	logger utils.Logger,
	cfg *config.Config,
) *grpc.Server {
	// Initialize admin service
	adminService := service.NewAdminService(db, redisClient, logger, cfg)

	// Create server instance
	server := &AdminServiceServer{
		db:           db,
		redisClient:  redisClient,
		adminService: adminService,
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
	// grpcServer.RegisterService(&proto.AdminService_ServiceDesc, server)

	return grpcServer
}

// unaryInterceptor provides logging and metrics for unary RPC calls
func (s *AdminServiceServer) unaryInterceptor(
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
	s.requestDuration.WithLabelValues(info.FullMethod, status.Code(err).String()).Observe(duration.Seconds())
	s.requestTotal.WithLabelValues(info.FullMethod, status.Code(err).String()).Inc()

	// Log the response
	if err != nil {
		s.logger.Error("gRPC request failed", "method", info.FullMethod, "error", err, "duration", duration)
	} else {
		s.logger.Info("gRPC request completed", "method", info.FullMethod, "duration", duration)
	}

	return resp, err
}

// streamInterceptor provides logging and metrics for streaming RPC calls
func (s *AdminServiceServer) streamInterceptor(
	srv interface{},
	ss grpc.ServerStream,
	info *grpc.StreamServerInfo,
	handler grpc.StreamHandler,
) error {
	start := time.Now()

	s.logger.Info("gRPC stream request",
("method", info.FullMethod),
	)

	err := handler(srv, ss)

	duration := time.Since(start)
	s.requestDuration.WithLabelValues(info.FullMethod, status.Code(err).String()).Observe(duration.Seconds())
	s.requestTotal.WithLabelValues(info.FullMethod, status.Code(err).String()).Inc()

	if err != nil {
		s.logger.Error("gRPC stream request failed", "method", info.FullMethod, "error", err, "duration", duration)
	} else {
		s.logger.Info("gRPC stream request completed", "method", info.FullMethod, "duration", duration)
	}

	return err
}

// initMetrics initializes Prometheus metrics
func (s *AdminServiceServer) initMetrics() {
	s.requestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "admin_service_request_duration_seconds",
			Help:    "Duration of gRPC requests in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "status"},
	)

	s.requestTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "admin_service_requests_total",
			Help: "Total number of gRPC requests",
		},
		[]string{"method", "status"},
	)
}

// Health check method
func (s *AdminServiceServer) HealthCheck(ctx context.Context) error {
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
