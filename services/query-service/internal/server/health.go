package server

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
)

// healthServer wires the custom readiness logic into the standard gRPC health interface.
type healthServer struct {
	grpc_health_v1.UnimplementedHealthServer
	queryServer *QueryServiceServer
}

func newHealthServer(qs *QueryServiceServer) *healthServer {
	return &healthServer{queryServer: qs}
}

// Check runs the existing dependency checks (Postgres + Redis) before reporting status.
func (h *healthServer) Check(ctx context.Context, _ *grpc_health_v1.HealthCheckRequest) (*grpc_health_v1.HealthCheckResponse, error) {
	if err := h.queryServer.HealthCheck(ctx); err != nil {
		h.queryServer.logger.Error("Health check failed", "error", err)
		return &grpc_health_v1.HealthCheckResponse{Status: grpc_health_v1.HealthCheckResponse_NOT_SERVING}, status.Error(codes.Unavailable, err.Error())
	}

	return &grpc_health_v1.HealthCheckResponse{Status: grpc_health_v1.HealthCheckResponse_SERVING}, nil
}

// List returns a single snapshot of the query service health.
func (h *healthServer) List(ctx context.Context, _ *grpc_health_v1.HealthListRequest) (*grpc_health_v1.HealthListResponse, error) {
	resp, err := h.Check(ctx, &grpc_health_v1.HealthCheckRequest{})
	statusResp := resp
	if statusResp == nil {
		statusResp = &grpc_health_v1.HealthCheckResponse{Status: grpc_health_v1.HealthCheckResponse_NOT_SERVING}
	}

	list := &grpc_health_v1.HealthListResponse{
		Statuses: map[string]*grpc_health_v1.HealthCheckResponse{
			"query-service": statusResp,
		},
	}

	return list, err
}

// Watch emits a single snapshot of the health state to keep the implementation lightweight.
func (h *healthServer) Watch(req *grpc_health_v1.HealthCheckRequest, stream grpc.ServerStreamingServer[grpc_health_v1.HealthCheckResponse]) error {
	resp, err := h.Check(stream.Context(), req)
	if err != nil {
		if sendErr := stream.Send(&grpc_health_v1.HealthCheckResponse{Status: grpc_health_v1.HealthCheckResponse_NOT_SERVING}); sendErr != nil {
			return sendErr
		}
		return err
	}

	return stream.Send(resp)
}
