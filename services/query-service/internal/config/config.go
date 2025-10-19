package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds the configuration for the Query Service
type Config struct {
	// Server configuration
	Port int `json:"port"`

	// Database configuration
	DatabaseURL string `json:"database_url"`

	// Redis configuration
	RedisURL string `json:"redis_url"`

	// Cache configuration
	CacheTTL time.Duration `json:"cache_ttl"`

	// Logging configuration
	LogLevel  string `json:"log_level"`
	LogFormat string `json:"log_format"`

	// Query configuration
	MaxQueryLimit int `json:"max_query_limit"`
	DefaultLimit  int `json:"default_limit"`

	// Performance configuration
	MaxConcurrentQueries int `json:"max_concurrent_queries"`
	QueryTimeout         time.Duration `json:"query_timeout"`
}

// Load loads configuration from environment variables
func Load(configPath string) (*Config, error) {
	cfg := &Config{
		Port:                  getEnvInt("QUERY_SERVICE_PORT", 8081),
		DatabaseURL:           getEnvString("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/event_indexer?sslmode=disable"),
		RedisURL:              getEnvString("REDIS_URL", "redis://localhost:6379"),
		CacheTTL:              getEnvDuration("CACHE_TTL", 30*time.Second),
		LogLevel:              getEnvString("LOG_LEVEL", "info"),
		LogFormat:             getEnvString("LOG_FORMAT", "json"),
		MaxQueryLimit:         getEnvInt("MAX_QUERY_LIMIT", 1000),
		DefaultLimit:          getEnvInt("DEFAULT_LIMIT", 20),
		MaxConcurrentQueries:  getEnvInt("MAX_CONCURRENT_QUERIES", 100),
		QueryTimeout:          getEnvDuration("QUERY_TIMEOUT", 10*time.Second),
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
