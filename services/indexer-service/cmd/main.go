package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	
	"github.com/smart-contract-event-indexer/indexer-service/internal/blockchain"
	"github.com/smart-contract-event-indexer/indexer-service/internal/config"
	"github.com/smart-contract-event-indexer/indexer-service/internal/indexer"
	"github.com/smart-contract-event-indexer/indexer-service/internal/storage"
	"github.com/smart-contract-event-indexer/shared/utils"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}
	
	// Validate configuration
	if err := cfg.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "Invalid configuration: %v\n", err)
		os.Exit(1)
	}
	
	// Initialize logger
	logger := utils.NewLogger("indexer-service", cfg.LogLevel, cfg.LogFormat)
	logger.Info("Starting Indexer Service")
	logger.WithFields(map[string]interface{}{
		"rpc_endpoint":   cfg.RPCEndpoint,
		"poll_interval":  cfg.PollInterval,
		"batch_size":     cfg.BatchSize,
		"confirm_blocks": cfg.ConfirmBlocks,
	}).Info("Configuration loaded")
	
	// Create main context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	// Initialize database connection
	db, err := sqlx.Connect("postgres", cfg.DatabaseURL)
	if err != nil {
		logger.WithError(err).Fatal("Failed to connect to database")
	}
	defer db.Close()
	
	// Configure connection pool
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)
	
	logger.Info("Database connection established")
	
	// Test database connection
	if err := db.Ping(); err != nil {
		logger.WithError(err).Fatal("Failed to ping database")
	}
	
	// Initialize blockchain client
	client := blockchain.NewClient(cfg.RPCEndpoint, logger)
	if err := client.Connect(ctx); err != nil {
		logger.WithError(err).Fatal("Failed to connect to blockchain")
	}
	defer client.Close()
	
	// Initialize storage layers
	contractStorage := storage.NewContractStorage(db, logger)
	eventStorage := storage.NewEventStorage(db, logger)
	stateStorage := storage.NewStateStorage(db, logger)
	
	// Initialize indexer
	idx := indexer.NewIndexer(
		client,
		contractStorage,
		eventStorage,
		stateStorage,
		cfg.PollInterval,
		cfg.BatchSize,
		logger,
	)
	
	// Start health check server
	healthServer := startHealthCheckServer(cfg.HealthPort, logger, db, client)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		healthServer.Shutdown(ctx)
	}()
	
	// Start indexer in a goroutine
	errChan := make(chan error, 1)
	go func() {
		if err := idx.Start(ctx); err != nil && err != context.Canceled {
			errChan <- err
		}
	}()
	
	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	
	logger.Info("Indexer service is running. Press Ctrl+C to stop.")
	
	select {
	case sig := <-sigChan:
		logger.WithField("signal", sig.String()).Info("Shutdown signal received")
	case err := <-errChan:
		logger.WithError(err).Error("Indexer error")
	}
	
	// Graceful shutdown
	logger.Info("Shutting down gracefully...")
	cancel()
	
	// Wait a bit for goroutines to finish
	time.Sleep(2 * time.Second)
	
	logger.Info("Indexer service stopped")
}

// startHealthCheckServer starts an HTTP server for health checks
func startHealthCheckServer(port int, logger utils.Logger, db *sqlx.DB, client *blockchain.Client) *http.Server {
	mux := http.NewServeMux()
	
	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		
		// Check database
		if err := db.PingContext(ctx); err != nil {
			logger.WithError(err).Error("Database health check failed")
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(`{"status":"unhealthy","reason":"database connection failed"}`))
			return
		}
		
		// Check blockchain connection
		if err := client.HealthCheck(ctx); err != nil {
			logger.WithError(err).Error("Blockchain health check failed")
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(`{"status":"unhealthy","reason":"blockchain connection failed"}`))
			return
		}
		
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"healthy"}`))
	})
	
	// Readiness endpoint
	mux.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ready"}`))
	})
	
	// Liveness endpoint
	mux.HandleFunc("/live", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"alive"}`))
	})
	
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	
	go func() {
		logger.WithField("port", port).Info("Health check server started")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.WithError(err).Error("Health check server error")
		}
	}()
	
	return server
}

