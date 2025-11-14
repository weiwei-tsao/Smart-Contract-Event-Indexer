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
	"github.com/smart-contract-event-indexer/shared/models"
	protoapi "github.com/smart-contract-event-indexer/shared/proto"
	"github.com/smart-contract-event-indexer/shared/utils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// AdminServiceServer implements the gRPC AdminService
type AdminServiceServer struct {
	db           *sql.DB
	redisClient  *redis.Client
	adminService *service.AdminService
	logger       utils.Logger
	config       *config.Config

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

	// Register proto service
	protoapi.RegisterAdminServiceServer(grpcServer, server)

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

	s.logger.Info("gRPC stream request", "method", info.FullMethod)

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

// checkDependencies validates backing services.
func (s *AdminServiceServer) checkDependencies(ctx context.Context) error {
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

// --- proto handlers ---

func (s *AdminServiceServer) AddContract(ctx context.Context, req *protoapi.AddContractRequest) (*protoapi.AddContractResponse, error) {
	resp, err := s.adminService.AddContract(ctx, &service.AddContractRequest{
		Address:       req.Address,
		Name:          req.Name,
		ABI:           req.Abi,
		StartBlock:    req.StartBlock,
		ConfirmBlocks: req.ConfirmBlocks,
	})
	if err != nil {
		return nil, err
	}

	var contractProto *protoapi.Contract
	if contract, err := s.adminService.GetContract(ctx, req.Address); err == nil && contract != nil {
		contractProto = convertContract(contract)
	}

	return &protoapi.AddContractResponse{
		Success:  resp.Success,
		Contract: contractProto,
		IsNew:    resp.IsNew,
		Message:  resp.Message,
	}, nil
}

func (s *AdminServiceServer) RemoveContract(ctx context.Context, req *protoapi.RemoveContractRequest) (*protoapi.RemoveContractResponse, error) {
	resp, err := s.adminService.RemoveContract(ctx, &service.RemoveContractRequest{
		Address: req.Address,
	})
	if err != nil {
		return nil, err
	}
	return &protoapi.RemoveContractResponse{
		Success: resp.Success,
		Message: resp.Message,
	}, nil
}

func (s *AdminServiceServer) GetContract(ctx context.Context, req *protoapi.GetContractRequest) (*protoapi.Contract, error) {
	contract, err := s.adminService.GetContract(ctx, req.Address)
	if err != nil {
		return nil, err
	}
	if contract == nil {
		return nil, status.Error(codes.NotFound, "contract not found")
	}
	return convertContract(contract), nil
}

func (s *AdminServiceServer) ListContracts(ctx context.Context, req *protoapi.ListContractsRequest) (*protoapi.ListContractsResponse, error) {
	contracts, total, err := s.adminService.ListContracts(ctx, req.Limit, req.Offset)
	if err != nil {
		return nil, err
	}

	result := make([]*protoapi.Contract, 0, len(contracts))
	for _, c := range contracts {
		result = append(result, convertContract(c))
	}

	return &protoapi.ListContractsResponse{
		Contracts:  result,
		TotalCount: total,
	}, nil
}

func (s *AdminServiceServer) TriggerBackfill(ctx context.Context, req *protoapi.BackfillRequest) (*protoapi.BackfillResponse, error) {
	resp, err := s.adminService.TriggerBackfill(ctx, &service.BackfillRequest{
		Address:   req.ContractAddress,
		FromBlock: req.FromBlock,
		ToBlock:   req.ToBlock,
	})
	if err != nil {
		return nil, err
	}
	return &protoapi.BackfillResponse{
		Success:       resp.Success,
		JobId:         resp.JobID,
		EstimatedTime: resp.EstimatedMinutes,
		Message:       resp.Message,
	}, nil
}

func (s *AdminServiceServer) GetBackfillStatus(ctx context.Context, req *protoapi.BackfillStatusRequest) (*protoapi.BackfillJob, error) {
	job, err := s.adminService.GetBackfillJob(ctx, req.JobId)
	if err != nil {
		return nil, err
	}
	if job == nil {
		return nil, status.Error(codes.NotFound, "job not found")
	}
	return convertBackfillJob(job), nil
}

func (s *AdminServiceServer) GetSystemStatus(ctx context.Context, _ *protoapi.Empty) (*protoapi.SystemStatusResponse, error) {
	statusResp, err := s.adminService.GetSystemStatus(ctx)
	if err != nil {
		return nil, err
	}

	services := make([]*protoapi.ServiceStatus, 0, len(statusResp.Services))
	for name, svc := range statusResp.Services {
		services = append(services, &protoapi.ServiceStatus{
			Name:    name,
			Status:  svc.Status,
			Latency: svc.Latency,
		})
	}

	return &protoapi.SystemStatusResponse{
		IndexerLag:       statusResp.IndexerLag,
		TotalContracts:   statusResp.TotalContracts,
		TotalEvents:      statusResp.TotalEvents,
		CacheHitRate:     statusResp.CacheHitRate,
		LastIndexedBlock: statusResp.LastIndexedBlock,
		IsHealthy:        statusResp.IsHealthy,
		Uptime:           statusResp.Uptime,
		Services:         services,
	}, nil
}

// HealthCheck satisfies the proto interface.
func (s *AdminServiceServer) HealthCheck(ctx context.Context, _ *protoapi.Empty) (*protoapi.HealthCheckResponse, error) {
	if err := s.checkDependencies(ctx); err != nil {
		return nil, err
	}
	now := timestamppb.Now()
	return &protoapi.HealthCheckResponse{
		Status:    "healthy",
		Timestamp: now,
	}, nil
}

// Helper conversions
func convertContract(contract *models.Contract) *protoapi.Contract {
	if contract == nil {
		return nil
	}
	return &protoapi.Contract{
		Id:            contract.ID,
		Address:       string(contract.Address),
		Abi:           contract.ABI,
		Name:          contract.Name,
		StartBlock:    contract.StartBlock,
		CurrentBlock:  contract.CurrentBlock,
		ConfirmBlocks: int32(contract.ConfirmBlocks),
		CreatedAt:     timestampOrNil(contract.CreatedAt),
		UpdatedAt:     timestampOrNil(contract.UpdatedAt),
	}
}

func convertBackfillJob(job *service.BackfillJob) *protoapi.BackfillJob {
	if job == nil {
		return nil
	}

	return &protoapi.BackfillJob{
		Id:              job.ID,
		ContractAddress: job.ContractAddress,
		FromBlock:       job.FromBlock,
		ToBlock:         job.ToBlock,
		CurrentBlock:    job.CurrentBlock,
		Status:          job.Status,
		ErrorMessage:    job.ErrorMessage,
		Progress:        job.Progress,
		CreatedAt:       timestampOrNil(job.CreatedAt),
		UpdatedAt:       timestampOrNil(job.UpdatedAt),
		CompletedAt: func() *timestamppb.Timestamp {
			if job.CompletedAt == nil || job.CompletedAt.IsZero() {
				return nil
			}
			return timestamppb.New(*job.CompletedAt)
		}(),
	}
}

func timestampOrNil(t time.Time) *timestamppb.Timestamp {
	if t.IsZero() {
		return nil
	}
	return timestamppb.New(t)
}
