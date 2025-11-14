package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/smart-contract-event-indexer/query-service/internal/config"
	"github.com/smart-contract-event-indexer/query-service/internal/server"
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
	logger := utils.NewLogger("query-service", cfg.LogLevel, cfg.LogFormat)
	logger.Info("Starting Query Service", "version", "1.0.0")
	logRuntimeConfig(logger, cfg, *configPath)

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

// logRuntimeConfig emits a structured summary of the boot configuration plus the ENV knobs ops should set.
func logRuntimeConfig(logger utils.Logger, cfg *config.Config, configPath string) {
	logger.Info("Configuration loaded",
		"port", cfg.Port,
		"config_file", configPath,
		"database_url", maskConnectionURL(cfg.DatabaseURL),
		"redis_url", maskConnectionURL(cfg.RedisURL),
		"query_timeout", cfg.QueryTimeout.String(),
		"slow_query_threshold", cfg.SlowQueryThreshold.String(),
		"default_limit", cfg.DefaultLimit,
		"max_limit", cfg.MaxQueryLimit,
		"cache_ttl", cfg.CacheTTL.String(),
		"aggregation_cache_ttl", cfg.AggregationCacheTTL.String(),
	)

	envDocs := []struct {
		Name        string
		Default     string
		Description string
	}{
		{"QUERY_SERVICE_PORT", fmt.Sprintf("%d", cfg.Port), "Port the gRPC server listens on"},
		{"DATABASE_URL", cfg.DatabaseURL, "PostgreSQL DSN used by the query layer"},
		{"REDIS_URL", cfg.RedisURL, "Redis connection URL for cache + rate limit data"},
		{"CACHE_TTL", cfg.CacheTTL.String(), "Default TTL applied to cached event queries"},
		{"AGGREGATION_CACHE_TTL", cfg.AggregationCacheTTL.String(), "TTL for stats/aggregation cache entries"},
		{"QUERY_TIMEOUT", cfg.QueryTimeout.String(), "Per-query timeout enforced on DB operations"},
		{"SLOW_QUERY_THRESHOLD", cfg.SlowQueryThreshold.String(), "Latency threshold that triggers slow-query logs"},
		{"MAX_QUERY_LIMIT", fmt.Sprintf("%d", cfg.MaxQueryLimit), "Ceiling for user-requested page sizes"},
		{"DEFAULT_LIMIT", fmt.Sprintf("%d", cfg.DefaultLimit), "Fallback page size when a client omits limits"},
		{"NEGATIVE_CACHE_TTL", cfg.NegativeCacheTTL.String(), "TTL for empty-result sentinels"},
	}

	for _, doc := range envDocs {
		logger.Info("Environment variable",
			"name", doc.Name,
			"default", doc.Default,
			"description", doc.Description,
		)
	}
}

func maskConnectionURL(raw string) string {
	if raw == "" {
		return raw
	}

	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}

	if u.User != nil {
		username := u.User.Username()
		if username != "" {
			u.User = url.User(username)
		}
	}

	if strings.Contains(u.Host, "@") {
		parts := strings.Split(u.Host, "@")
		u.Host = parts[len(parts)-1]
	}

	return u.String()
}
