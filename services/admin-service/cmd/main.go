package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/smart-contract-event-indexer/admin-service/internal/config"
	"github.com/smart-contract-event-indexer/admin-service/internal/server"
	"github.com/smart-contract-event-indexer/shared/database"
	"github.com/smart-contract-event-indexer/shared/utils"
	"go.uber.org/zap"
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
	logger, err := utils.NewLogger(cfg.LogLevel, cfg.LogFormat)
	if err != nil {
		log.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Sync()

	logger.Info("Starting Admin Service", zap.String("version", "1.0.0"))

	// Initialize database connection
	db, err := database.NewConnection(cfg.DatabaseURL)
	if err != nil {
		logger.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer db.Close()

	// Initialize Redis connection
	redisClient, err := database.NewRedisClient(cfg.RedisURL)
	if err != nil {
		logger.Fatal("Failed to connect to Redis", zap.Error(err))
	}
	defer redisClient.Close()

	// Create and start gRPC server
	grpcServer := server.NewAdminServiceServer(db, redisClient, logger, cfg)

	// Start server in a goroutine
	go func() {
		lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.Port))
		if err != nil {
			logger.Fatal("Failed to listen", zap.Error(err))
		}

		logger.Info("Admin Service started", zap.String("address", lis.Addr().String()))
		if err := grpcServer.Serve(lis); err != nil {
			logger.Fatal("Failed to serve", zap.Error(err))
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down Admin Service...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	grpcServer.GracefulStop()

	logger.Info("Admin Service stopped")
}
