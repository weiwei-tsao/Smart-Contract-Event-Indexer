package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds the configuration for the Admin Service
type Config struct {
	// Server configuration
	Port int `json:"port"`

	// Database configuration
	DatabaseURL string `json:"database_url"`

	// Redis configuration
	RedisURL string `json:"redis_url"`

	// Logging configuration
	LogLevel  string `json:"log_level"`
	LogFormat string `json:"log_format"`

	// Backfill configuration
	ChunkSize           int           `json:"chunk_size"`
	MaxConcurrentChunks int           `json:"max_concurrent_chunks"`
	RateLimit           int           `json:"rate_limit"`
	BackfillTimeout     time.Duration `json:"backfill_timeout"`
}

// Load loads configuration from environment variables
func Load(configPath string) (*Config, error) {
	cfg := &Config{
		Port:                 getEnvInt("ADMIN_SERVICE_PORT", 8082),
		DatabaseURL:          getEnvString("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/event_indexer?sslmode=disable"),
		RedisURL:             getEnvString("REDIS_URL", "redis://localhost:6379"),
		LogLevel:             getEnvString("LOG_LEVEL", "info"),
		LogFormat:            getEnvString("LOG_FORMAT", "json"),
		ChunkSize:            getEnvInt("CHUNK_SIZE", 1000),
		MaxConcurrentChunks:  getEnvInt("MAX_CONCURRENT_CHUNKS", 3),
		RateLimit:            getEnvInt("RATE_LIMIT", 100),
		BackfillTimeout:      getEnvDuration("BACKFILL_TIMEOUT", 30*time.Minute),
	}

	return cfg, nil
}

// getEnvString gets an environment variable as a string with a default value
func getEnvString(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvInt gets an environment variable as an integer with a default value
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// getEnvDuration gets an environment variable as a duration with a default value
func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}
