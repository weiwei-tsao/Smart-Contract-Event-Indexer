package config

import (
	"os"
	"strconv"
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
	QueryServiceAddr string `json:"query_service_addr"`
	AdminServiceAddr string `json:"admin_service_addr"`
	GRPCTimeout      time.Duration `json:"grpc_timeout"`

	// CORS configuration
	CORSOrigins []string `json:"cors_origins"`

	// Rate limiting configuration
	RateLimitFreeTier int `json:"rate_limit_free_tier"`
	RateLimitProTier  int `json:"rate_limit_pro_tier"`

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
		CORSOrigins:       getEnvStringSlice("CORS_ORIGINS", []string{"http://localhost:3000"}),
		RateLimitFreeTier: getEnvInt("RATE_LIMIT_FREE_TIER", 100),
		RateLimitProTier:  getEnvInt("RATE_LIMIT_PRO_TIER", 1000),
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
		// Simple comma-separated parsing
		parts := []string{}
		for _, part := range []string{value} {
			if part != "" {
				parts = append(parts, part)
			}
		}
		if len(parts) > 0 {
			return parts
		}
	}
	return defaultValue
}
