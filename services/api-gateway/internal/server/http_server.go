package server

import (
	"database/sql"
	"fmt"
	"net/http"
	"time"

	gqlhandler "github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/smart-contract-event-indexer/api-gateway/graph"
	"github.com/smart-contract-event-indexer/api-gateway/graph/generated"
	"github.com/smart-contract-event-indexer/api-gateway/internal/config"
	"github.com/smart-contract-event-indexer/api-gateway/internal/handler"
	"github.com/smart-contract-event-indexer/api-gateway/internal/middleware"
	protoapi "github.com/smart-contract-event-indexer/shared/proto"
	"github.com/smart-contract-event-indexer/shared/utils"
)

// HTTPServer handles HTTP requests
type HTTPServer struct {
	server *http.Server
	logger utils.Logger
}

// NewHTTPServer creates a new HTTP server
func NewHTTPServer(
	db *sql.DB,
	redisClient *redis.Client,
	queryClient protoapi.QueryServiceClient,
	adminClient protoapi.AdminServiceClient,
	logger utils.Logger,
	cfg *config.Config,
) *http.Server {
	// Set Gin mode
	if cfg.LogLevel == "debug" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	// Create Gin router
	router := gin.New()

	// Add middleware
	router.Use(middleware.Logger(logger))
	router.Use(middleware.Recovery(logger))
	router.Use(middleware.CORS(cfg.CORSOrigins))
	router.Use(middleware.RateLimiter(redisClient, cfg.RateLimitFreeTier, time.Minute))

	// Create handlers
	eventHandler := handler.NewEventHandler(db, redisClient, queryClient, logger, cfg)
	contractHandler := handler.NewContractHandler(db, redisClient, adminClient, queryClient, logger, cfg)
	healthHandler := handler.NewHealthHandler(db, redisClient, logger)

	// GraphQL resolver
	resolver := &graph.Resolver{
		DB:          db,
		Redis:       redisClient,
		QueryClient: queryClient,
		AdminClient: adminClient,
		Logger:      logger,
		Config:      cfg,
	}
	gqlServer := gqlhandler.NewDefaultServer(
		generated.NewExecutableSchema(generated.Config{Resolvers: resolver}),
	)

	// API routes
	api := router.Group("/api/v1")
	{
		// Event routes
		events := api.Group("/events")
		{
			events.GET("", eventHandler.GetEvents)
			events.GET("/tx/:txHash", eventHandler.GetEventsByTransaction)
			events.GET("/address/:address", eventHandler.GetEventsByAddress)
		}

		// Contract routes
		contracts := api.Group("/contracts")
		{
			contracts.GET("", contractHandler.GetContracts)
			contracts.POST("", contractHandler.AddContract)
			contracts.GET("/:address", contractHandler.GetContract)
			contracts.DELETE("/:address", contractHandler.RemoveContract)
			contracts.GET("/:address/stats", contractHandler.GetContractStats)
		}

		// Health check
		api.GET("/health", healthHandler.HealthCheck)
	}

	// GraphQL endpoints
	router.POST("/graphql", func(c *gin.Context) {
		gqlServer.ServeHTTP(c.Writer, c.Request)
	})
	router.GET("/graphql", func(c *gin.Context) {
		gqlServer.ServeHTTP(c.Writer, c.Request)
	})
	router.GET("/playground", func(c *gin.Context) {
		playground.Handler("GraphQL Playground", "/graphql").ServeHTTP(c.Writer, c.Request)
	})

	// Create HTTP server
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	return server
}
