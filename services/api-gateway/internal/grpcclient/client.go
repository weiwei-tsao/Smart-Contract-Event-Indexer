package grpcclient

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/smart-contract-event-indexer/api-gateway/internal/config"
	protoapi "github.com/smart-contract-event-indexer/shared/proto"
	"github.com/smart-contract-event-indexer/shared/utils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Clients bundles outbound gRPC clients with resiliency.
type Clients struct {
	Query protoapi.QueryServiceClient
	Admin protoapi.AdminServiceClient

	queryPool *grpcPool[protoapi.QueryServiceClient]
	adminPool *grpcPool[protoapi.AdminServiceClient]
}

// Close tears down pooled connections.
func (c *Clients) Close() {
	if c.queryPool != nil {
		c.queryPool.close()
	}
	if c.adminPool != nil {
		c.adminPool.close()
	}
}

// NewClients dials downstream services with connection pooling.
func NewClients(cfg *config.Config, logger utils.Logger) (*Clients, error) {
	poolSize := cfg.GRPCPoolSize
	if poolSize < 1 {
		poolSize = 1
	}

	queryPool, err := newPool(cfg.QueryServiceAddr, poolSize, cfg.GRPCTimeout, func(conn *grpc.ClientConn) protoapi.QueryServiceClient {
		return protoapi.NewQueryServiceClient(conn)
	})
	if err != nil {
		return nil, fmt.Errorf("query service dial: %w", err)
	}
	logger.Info("Connected to query service", "addr", cfg.QueryServiceAddr, "pool", poolSize)

	adminPool, err := newPool(cfg.AdminServiceAddr, poolSize, cfg.GRPCTimeout, func(conn *grpc.ClientConn) protoapi.AdminServiceClient {
		return protoapi.NewAdminServiceClient(conn)
	})
	if err != nil {
		queryPool.close()
		return nil, fmt.Errorf("admin service dial: %w", err)
	}
	logger.Info("Connected to admin service", "addr", cfg.AdminServiceAddr, "pool", poolSize)

	retries := cfg.GRPCRetries
	if retries < 1 {
		retries = 1
	}
	backoff := cfg.GRPCRetryBackoff
	if backoff <= 0 {
		backoff = 100 * time.Millisecond
	}

	return &Clients{
		Query:     &resilientQueryClient{pool: queryPool, retries: retries, backoff: backoff},
		Admin:     &resilientAdminClient{pool: adminPool, retries: retries, backoff: backoff},
		queryPool: queryPool,
		adminPool: adminPool,
	}, nil
}

type grpcPool[T any] struct {
	conns   []*grpc.ClientConn
	clients []T
	counter uint64
}

func newPool[T any](target string, size int, timeout time.Duration, factory func(conn *grpc.ClientConn) T) (*grpcPool[T], error) {
	conns := make([]*grpc.ClientConn, 0, size)
	clients := make([]T, 0, size)

	for i := 0; i < size; i++ {
		conn, err := dial(target, timeout)
		if err != nil {
			for _, c := range conns {
				_ = c.Close()
			}
			return nil, err
		}
		conns = append(conns, conn)
		clients = append(clients, factory(conn))
	}

	return &grpcPool[T]{conns: conns, clients: clients}, nil
}

func (p *grpcPool[T]) pick() T {
	index := atomic.AddUint64(&p.counter, 1)
	return p.clients[int((index-1)%uint64(len(p.clients)))]
}

func (p *grpcPool[T]) close() {
	for _, conn := range p.conns {
		_ = conn.Close()
	}
}

func dial(target string, timeout time.Duration) (*grpc.ClientConn, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	conn, err := grpc.DialContext(
		ctx,
		target,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

type resilientQueryClient struct {
	pool    *grpcPool[protoapi.QueryServiceClient]
	retries int
	backoff time.Duration
}

func (c *resilientQueryClient) GetEvents(ctx context.Context, in *protoapi.EventQuery, opts ...grpc.CallOption) (*protoapi.EventResponse, error) {
	call := func(client protoapi.QueryServiceClient) (*protoapi.EventResponse, error) {
		return client.GetEvents(ctx, in, opts...)
	}
	return retry(ctx, c.pool, c.retries, c.backoff, call)
}

func (c *resilientQueryClient) GetEventsByAddress(ctx context.Context, in *protoapi.AddressQuery, opts ...grpc.CallOption) (*protoapi.EventResponse, error) {
	call := func(client protoapi.QueryServiceClient) (*protoapi.EventResponse, error) {
		return client.GetEventsByAddress(ctx, in, opts...)
	}
	return retry(ctx, c.pool, c.retries, c.backoff, call)
}

func (c *resilientQueryClient) GetEventsByTransaction(ctx context.Context, in *protoapi.TransactionQuery, opts ...grpc.CallOption) (*protoapi.EventResponse, error) {
	call := func(client protoapi.QueryServiceClient) (*protoapi.EventResponse, error) {
		return client.GetEventsByTransaction(ctx, in, opts...)
	}
	return retry(ctx, c.pool, c.retries, c.backoff, call)
}

func (c *resilientQueryClient) GetContractStats(ctx context.Context, in *protoapi.StatsQuery, opts ...grpc.CallOption) (*protoapi.StatsResponse, error) {
	call := func(client protoapi.QueryServiceClient) (*protoapi.StatsResponse, error) {
		return client.GetContractStats(ctx, in, opts...)
	}
	return retry(ctx, c.pool, c.retries, c.backoff, call)
}

type resilientAdminClient struct {
	pool    *grpcPool[protoapi.AdminServiceClient]
	retries int
	backoff time.Duration
}

func (c *resilientAdminClient) AddContract(ctx context.Context, in *protoapi.AddContractRequest, opts ...grpc.CallOption) (*protoapi.AddContractResponse, error) {
	call := func(client protoapi.AdminServiceClient) (*protoapi.AddContractResponse, error) {
		return client.AddContract(ctx, in, opts...)
	}
	return retry(ctx, c.pool, c.retries, c.backoff, call)
}

func (c *resilientAdminClient) RemoveContract(ctx context.Context, in *protoapi.RemoveContractRequest, opts ...grpc.CallOption) (*protoapi.RemoveContractResponse, error) {
	call := func(client protoapi.AdminServiceClient) (*protoapi.RemoveContractResponse, error) {
		return client.RemoveContract(ctx, in, opts...)
	}
	return retry(ctx, c.pool, c.retries, c.backoff, call)
}

func (c *resilientAdminClient) GetContract(ctx context.Context, in *protoapi.GetContractRequest, opts ...grpc.CallOption) (*protoapi.Contract, error) {
	call := func(client protoapi.AdminServiceClient) (*protoapi.Contract, error) {
		return client.GetContract(ctx, in, opts...)
	}
	return retry(ctx, c.pool, c.retries, c.backoff, call)
}

func (c *resilientAdminClient) ListContracts(ctx context.Context, in *protoapi.ListContractsRequest, opts ...grpc.CallOption) (*protoapi.ListContractsResponse, error) {
	call := func(client protoapi.AdminServiceClient) (*protoapi.ListContractsResponse, error) {
		return client.ListContracts(ctx, in, opts...)
	}
	return retry(ctx, c.pool, c.retries, c.backoff, call)
}

func (c *resilientAdminClient) TriggerBackfill(ctx context.Context, in *protoapi.BackfillRequest, opts ...grpc.CallOption) (*protoapi.BackfillResponse, error) {
	call := func(client protoapi.AdminServiceClient) (*protoapi.BackfillResponse, error) {
		return client.TriggerBackfill(ctx, in, opts...)
	}
	return retry(ctx, c.pool, c.retries, c.backoff, call)
}

func (c *resilientAdminClient) GetBackfillStatus(ctx context.Context, in *protoapi.BackfillStatusRequest, opts ...grpc.CallOption) (*protoapi.BackfillJob, error) {
	call := func(client protoapi.AdminServiceClient) (*protoapi.BackfillJob, error) {
		return client.GetBackfillStatus(ctx, in, opts...)
	}
	return retry(ctx, c.pool, c.retries, c.backoff, call)
}

func (c *resilientAdminClient) GetSystemStatus(ctx context.Context, in *protoapi.Empty, opts ...grpc.CallOption) (*protoapi.SystemStatusResponse, error) {
	call := func(client protoapi.AdminServiceClient) (*protoapi.SystemStatusResponse, error) {
		return client.GetSystemStatus(ctx, in, opts...)
	}
	return retry(ctx, c.pool, c.retries, c.backoff, call)
}

func (c *resilientAdminClient) HealthCheck(ctx context.Context, in *protoapi.Empty, opts ...grpc.CallOption) (*protoapi.HealthCheckResponse, error) {
	call := func(client protoapi.AdminServiceClient) (*protoapi.HealthCheckResponse, error) {
		return client.HealthCheck(ctx, in, opts...)
	}
	return retry(ctx, c.pool, c.retries, c.backoff, call)
}

func retry[T any, C interface{}](ctx context.Context, pool *grpcPool[C], retries int, backoff time.Duration, call func(client C) (T, error)) (T, error) {
	var zero T
	var lastErr error
	for attempt := 0; attempt < retries; attempt++ {
		client := pool.pick()
		resp, err := call(client)
		if err == nil {
			return resp, nil
		}
		lastErr = err
		if ctx.Err() != nil {
			return zero, ctx.Err()
		}
		time.Sleep(backoff * time.Duration(attempt+1))
	}
	return zero, lastErr
}
