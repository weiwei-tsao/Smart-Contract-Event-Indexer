package grpcclient

import (
	"context"
	"fmt"

	"github.com/smart-contract-event-indexer/api-gateway/internal/config"
	protoapi "github.com/smart-contract-event-indexer/shared/proto"
	"github.com/smart-contract-event-indexer/shared/utils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Clients bundles outbound gRPC clients.
type Clients struct {
	Query protoapi.QueryServiceClient
	Admin protoapi.AdminServiceClient

	queryConn *grpc.ClientConn
	adminConn *grpc.ClientConn
}

// Close tears down all connections.
func (c *Clients) Close() {
	if c.queryConn != nil {
		_ = c.queryConn.Close()
	}
	if c.adminConn != nil {
		_ = c.adminConn.Close()
	}
}

// NewClients dials the downstream gRPC services.
func NewClients(cfg *config.Config, logger utils.Logger) (*Clients, error) {
	dial := func(target string) (*grpc.ClientConn, error) {
		ctx, cancel := context.WithTimeout(context.Background(), cfg.GRPCTimeout)
		defer cancel()

		conn, err := grpc.DialContext(
			ctx,
			target,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithBlock(),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to dial %s: %w", target, err)
		}
		return conn, nil
	}

	queryConn, err := dial(cfg.QueryServiceAddr)
	if err != nil {
		return nil, err
	}
	logger.Info("Connected to query service", "addr", cfg.QueryServiceAddr)

	adminConn, err := dial(cfg.AdminServiceAddr)
	if err != nil {
		_ = queryConn.Close()
		return nil, err
	}
	logger.Info("Connected to admin service", "addr", cfg.AdminServiceAddr)

	return &Clients{
		Query:     protoapi.NewQueryServiceClient(queryConn),
		Admin:     protoapi.NewAdminServiceClient(adminConn),
		queryConn: queryConn,
		adminConn: adminConn,
	}, nil
}
