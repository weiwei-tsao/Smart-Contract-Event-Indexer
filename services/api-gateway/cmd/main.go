package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/smart-contract-event-indexer/api-gateway/internal/config"
	"github.com/smart-contract-event-indexer/api-gateway/internal/server"
	sharedconfig "github.com/smart-contract-event-indexer/shared/config"
	"github.com/smart-contract-event-indexer/shared/database"
	"github.com/smart-contract-event-indexer/shared/utils"
)

func main() {
	var configPath = flag.String("config", "", "path to config file")
	flag.Parse()

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize logger
	logger := utils.NewLogger("api-gateway", cfg.LogLevel, cfg.LogFormat)
	logger.Info("Starting API Gateway", "version", "1.0.0")

	// Initialize database connection
	dbConfig := sharedconfig.DatabaseConfig{
		URL:             cfg.DatabaseURL,
		MaxOpenConns:    20,
		MaxIdleConns:    5,
		ConnMaxLifetime: 5 * time.Minute,
	}
	db, err := database.NewDB(dbConfig, logger)
	if err != nil {
		logger.Fatal("Failed to connect to database", "error", err)
	}
	defer db.Close()

	// Initialize Redis connection
	redisConfig := sharedconfig.RedisConfig{
		URL:      cfg.RedisURL,
		Password: "",
		DB:       0,
	}
	redisClient, err := database.NewRedisClient(redisConfig, logger)
	if err != nil {
		logger.Fatal("Failed to connect to Redis", "error", err)
	}
	defer redisClient.Close()

	// Create and start HTTP server
	httpServer := server.NewHTTPServer(db.DB, redisClient.Client, logger, cfg)

	// Start server in a goroutine
	go func() {
		logger.Info("API Gateway started", "address", fmt.Sprintf(":%d", cfg.Port))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to start HTTP server", "error", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down API Gateway...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown", "error", err)
	}

	logger.Info("API Gateway stopped")
}
