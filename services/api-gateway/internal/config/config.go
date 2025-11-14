package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds the configuration for the API Gateway
type Config struct {
	// Server configuration
	Port int `json:"port"`

	// Database configuration
	DatabaseURL string `json:"database_url"`

	// Redis configuration
	RedisURL string `json:"redis_url"`

	// gRPC configuration
	QueryServiceAddr string        `json:"query_service_addr"`
	AdminServiceAddr string        `json:"admin_service_addr"`
	GRPCTimeout      time.Duration `json:"grpc_timeout"`
	GRPCPoolSize     int           `json:"grpc_pool_size"`
	GRPCRetries      int           `json:"grpc_retries"`
	GRPCRetryBackoff time.Duration `json:"grpc_retry_backoff"`

	// CORS configuration
	CORSOrigins []string `json:"cors_origins"`

	// Rate limiting configuration
	RateLimitFreeTier int      `json:"rate_limit_free_tier"`
	RateLimitProTier  int      `json:"rate_limit_pro_tier"`
	APIKeysFree       []string `json:"api_keys_free"`
	APIKeysPro        []string `json:"api_keys_pro"`

	// Logging configuration
	LogLevel  string `json:"log_level"`
	LogFormat string `json:"log_format"`

	// API configuration
	MaxQueryLimit int `json:"max_query_limit"`
	DefaultLimit  int `json:"default_limit"`
}

// Load loads configuration from environment variables
func Load(configPath string) (*Config, error) {
	cfg := &Config{
		Port:              getEnvInt("API_GATEWAY_PORT", 8000),
		DatabaseURL:       getEnvString("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/event_indexer?sslmode=disable"),
		RedisURL:          getEnvString("REDIS_URL", "redis://localhost:6379"),
		QueryServiceAddr:  getEnvString("QUERY_SERVICE_ADDR", "localhost:8081"),
		AdminServiceAddr:  getEnvString("ADMIN_SERVICE_ADDR", "localhost:8082"),
		GRPCTimeout:       getEnvDuration("GRPC_TIMEOUT", 10*time.Second),
		GRPCPoolSize:      getEnvInt("GRPC_POOL_SIZE", 2),
		GRPCRetries:       getEnvInt("GRPC_RETRIES", 3),
		GRPCRetryBackoff:  getEnvDuration("GRPC_RETRY_BACKOFF", 200*time.Millisecond),
		CORSOrigins:       getEnvStringSlice("CORS_ORIGINS", []string{"http://localhost:3000"}),
		RateLimitFreeTier: getEnvInt("RATE_LIMIT_FREE_TIER", 100),
		RateLimitProTier:  getEnvInt("RATE_LIMIT_PRO_TIER", 1000),
		APIKeysFree:       getEnvCSV("API_KEYS_FREE"),
		APIKeysPro:        getEnvCSV("API_KEYS_PRO"),
		LogLevel:          getEnvString("LOG_LEVEL", "info"),
		LogFormat:         getEnvString("LOG_FORMAT", "json"),
		MaxQueryLimit:     getEnvInt("MAX_QUERY_LIMIT", 1000),
		DefaultLimit:      getEnvInt("DEFAULT_LIMIT", 20),
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

// getEnvStringSlice gets an environment variable as a string slice with a default value
func getEnvStringSlice(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		parts := strings.Split(value, ",")
		out := make([]string, 0, len(parts))
		for _, part := range parts {
			if trimmed := strings.TrimSpace(part); trimmed != "" {
				out = append(out, trimmed)
			}
		}
		if len(out) > 0 {
			return out
		}
	}
	return defaultValue
}

func getEnvCSV(key string) []string {
	value := os.Getenv(key)
	if value == "" {
		return nil
	}
	return getEnvStringSlice(key, nil)
}
