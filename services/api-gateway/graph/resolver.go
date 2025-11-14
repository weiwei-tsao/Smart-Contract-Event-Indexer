package graph

import (
	"database/sql"

	"github.com/redis/go-redis/v9"
	"github.com/smart-contract-event-indexer/api-gateway/internal/config"
	protoapi "github.com/smart-contract-event-indexer/shared/proto"
	"github.com/smart-contract-event-indexer/shared/utils"
)

// Resolver wires dependencies for gqlgen.
type Resolver struct {
	DB          *sql.DB
	Redis       *redis.Client
	QueryClient protoapi.QueryServiceClient
	AdminClient protoapi.AdminServiceClient
	Logger      utils.Logger
	Config      *config.Config
}
