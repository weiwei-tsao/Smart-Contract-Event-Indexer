package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds all configuration for the indexer service
type Config struct {
	// RPC Configuration
	RPCEndpoint  string
	RPCFallbacks []string

	// Database Configuration
	DatabaseURL string
	RedisURL    string

	// Indexer Settings
	PollInterval     time.Duration
	BatchSize        int
	ConfirmBlocks    int
	MaxRetries       int
	RetryDelay       time.Duration
	MaxConcurrent    int

	// Logging
	LogLevel  string
	LogFormat string

	// Server
	Port         int
	MetricsPort  int
	HealthPort   int
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	cfg := &Config{
		// RPC defaults
		RPCEndpoint: getEnvOrDefault("RPC_ENDPOINT", "http://localhost:8545"),
		RPCFallbacks: []string{},

		// Database defaults
		DatabaseURL: os.Getenv("DATABASE_URL"),
		RedisURL:    getEnvOrDefault("REDIS_URL", "redis://localhost:6379"),

		// Indexer defaults
		PollInterval:  parseDurationOrDefault("POLL_INTERVAL", 6*time.Second),
		BatchSize:     parseIntOrDefault("BATCH_SIZE", 100),
		ConfirmBlocks: parseIntOrDefault("CONFIRM_BLOCKS", 6),
		MaxRetries:    parseIntOrDefault("MAX_RETRIES", 3),
		RetryDelay:    parseDurationOrDefault("RETRY_DELAY", 5*time.Second),
		MaxConcurrent: parseIntOrDefault("MAX_CONCURRENT_CONTRACTS", 5),

		// Logging defaults
		LogLevel:  getEnvOrDefault("LOG_LEVEL", "info"),
		LogFormat: getEnvOrDefault("LOG_FORMAT", "json"),

		// Server defaults
		Port:        parseIntOrDefault("PORT", 8080),
		MetricsPort: parseIntOrDefault("METRICS_PORT", 9090),
		HealthPort:  parseIntOrDefault("HEALTH_PORT", 8081),
	}

	// Parse fallback RPC endpoints if provided
	if fallbacks := os.Getenv("RPC_FALLBACKS"); fallbacks != "" {
		// Simple comma-separated parsing
		// In production, use proper config parser
		cfg.RPCFallbacks = []string{fallbacks}
	}

	// Validate required fields
	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	return cfg, nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.RPCEndpoint == "" {
		return fmt.Errorf("RPC_ENDPOINT cannot be empty")
	}
	if c.DatabaseURL == "" {
		return fmt.Errorf("DATABASE_URL cannot be empty")
	}
	if c.PollInterval <= 0 {
		return fmt.Errorf("POLL_INTERVAL must be positive")
	}
	if c.BatchSize <= 0 {
		return fmt.Errorf("BATCH_SIZE must be positive")
	}
	if c.ConfirmBlocks < 1 || c.ConfirmBlocks > 100 {
		return fmt.Errorf("CONFIRM_BLOCKS must be between 1 and 100")
	}
	return nil
}

// Helper functions

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func parseIntOrDefault(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

func parseDurationOrDefault(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if parsed, err := time.ParseDuration(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

