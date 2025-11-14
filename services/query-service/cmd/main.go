package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/smart-contract-event-indexer/query-service/internal/config"
	"github.com/smart-contract-event-indexer/query-service/internal/server"
	"github.com/smart-contract-event-indexer/shared/database"
	sharedconfig "github.com/smart-contract-event-indexer/shared/config"
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
	logger := utils.NewLogger("query-service", cfg.LogLevel, cfg.LogFormat)
	logger.Info("Starting Query Service", "version", "1.0.0")

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

	// Create and start gRPC server
	grpcServer := server.NewQueryServiceServer(db.DB, redisClient.Client, logger, cfg)

	// Start server in a goroutine
	go func() {
		lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.Port))
		if err != nil {
			logger.Fatal("Failed to listen", "error", err)
		}

		logger.Info("Query Service started", "address", lis.Addr().String())
		if err := grpcServer.Serve(lis); err != nil {
			logger.Fatal("Failed to serve", "error", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down Query Service...")

	// Graceful shutdown
	grpcServer.GracefulStop()

	logger.Info("Query Service stopped")
}
