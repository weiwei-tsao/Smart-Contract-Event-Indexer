package server

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/smart-contract-event-indexer/api-gateway/internal/config"
	"github.com/smart-contract-event-indexer/api-gateway/internal/handler"
	"github.com/smart-contract-event-indexer/api-gateway/internal/middleware"
	"github.com/smart-contract-event-indexer/shared/models"
	"go.uber.org/zap"
)

// HTTPServer handles HTTP requests
type HTTPServer struct {
	server *http.Server
	logger *zap.Logger
}

// NewHTTPServer creates a new HTTP server
func NewHTTPServer(
	db *sql.DB,
	redisClient *redis.Client,
	logger *zap.Logger,
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

	// Create handlers
	eventHandler := handler.NewEventHandler(db, redisClient, logger, cfg)
	contractHandler := handler.NewContractHandler(db, redisClient, logger, cfg)
	healthHandler := handler.NewHealthHandler(db, redisClient, logger)

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

	// GraphQL endpoint (placeholder)
	router.POST("/graphql", func(c *gin.Context) {
		c.JSON(http.StatusNotImplemented, gin.H{
			"error": "GraphQL endpoint not implemented yet",
		})
	})

	// GraphQL Playground (placeholder)
	router.GET("/playground", func(c *gin.Context) {
		c.HTML(http.StatusOK, "playground.html", gin.H{
			"title": "GraphQL Playground",
		})
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
