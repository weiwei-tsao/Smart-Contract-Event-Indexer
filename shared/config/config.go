package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	Database DatabaseConfig `yaml:"database"`
	Redis    RedisConfig    `yaml:"redis"`
	RPC      RPCConfig      `yaml:"rpc"`
	Indexer  IndexerConfig  `yaml:"indexer"`
	API      APIConfig      `yaml:"api"`
	Logging  LoggingConfig  `yaml:"logging"`
	Service  ServiceConfig  `yaml:"service"`
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	URL             string        `yaml:"url"`
	MaxOpenConns    int           `yaml:"max_open_conns"`
	MaxIdleConns    int           `yaml:"max_idle_conns"`
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime"`
}

// RedisConfig holds Redis configuration
type RedisConfig struct {
	URL      string `yaml:"url"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

// RPCConfig holds blockchain RPC configuration
type RPCConfig struct {
	Endpoint     string        `yaml:"endpoint"`
	Fallbacks    []string      `yaml:"fallbacks"`
	MaxRetry     int           `yaml:"max_retry"`
	RetryDelay   time.Duration `yaml:"retry_delay"`
}

// IndexerConfig holds indexer service configuration
type IndexerConfig struct {
	BatchSize              int           `yaml:"batch_size"`
	DefaultConfirmBlocks   int           `yaml:"default_confirm_blocks"`
	PollInterval           time.Duration `yaml:"poll_interval"`
	MaxConcurrentContracts int           `yaml:"max_concurrent_contracts"`
}

// APIConfig holds API service configuration
type APIConfig struct {
	Port            int      `yaml:"port"`
	CORSOrigins     []string `yaml:"cors_origins"`
	RateLimit       int      `yaml:"rate_limit"`
	EnablePlayground bool    `yaml:"enable_playground"`
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
}

// ServiceConfig holds service-specific configuration
type ServiceConfig struct {
	Name        string `yaml:"name"`
	Port        int    `yaml:"port"`
	Environment string `yaml:"environment"`
}

// LoadConfig loads configuration from environment variables and optional YAML file
func LoadConfig(yamlPath string) (*Config, error) {
	config := &Config{
		Database: DatabaseConfig{
			URL:             getEnv("DATABASE_URL", "postgres://indexer:indexer_password@localhost:5432/event_indexer?sslmode=disable"),
			MaxOpenConns:    getEnvAsInt("DB_MAX_OPEN_CONNS", 20),
			MaxIdleConns:    getEnvAsInt("DB_MAX_IDLE_CONNS", 5),
			ConnMaxLifetime: getEnvAsDuration("DB_CONN_MAX_LIFETIME", 5*time.Minute),
		},
		Redis: RedisConfig{
			URL:      getEnv("REDIS_URL", "redis://localhost:6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvAsInt("REDIS_DB", 0),
		},
		RPC: RPCConfig{
			Endpoint:   getEnv("RPC_ENDPOINT", "http://localhost:8545"),
			Fallbacks:  getEnvAsSlice("RPC_FALLBACKS", []string{}),
			MaxRetry:   getEnvAsInt("RPC_MAX_RETRY", 3),
			RetryDelay: getEnvAsDuration("RPC_RETRY_DELAY", 5*time.Second),
		},
		Indexer: IndexerConfig{
			BatchSize:              getEnvAsInt("INDEXER_BATCH_SIZE", 100),
			DefaultConfirmBlocks:   getEnvAsInt("INDEXER_DEFAULT_CONFIRM_BLOCKS", 6),
			PollInterval:           getEnvAsDuration("INDEXER_POLL_INTERVAL", 6*time.Second),
			MaxConcurrentContracts: getEnvAsInt("INDEXER_MAX_CONCURRENT_CONTRACTS", 5),
		},
		API: APIConfig{
			Port:             getEnvAsInt("API_GATEWAY_PORT", 8000),
			CORSOrigins:      getEnvAsSlice("API_CORS_ORIGINS", []string{"http://localhost:3000"}),
			RateLimit:        getEnvAsInt("API_RATE_LIMIT", 100),
			EnablePlayground: getEnvAsBool("API_ENABLE_PLAYGROUND", true),
		},
		Logging: LoggingConfig{
			Level:  getEnv("LOG_LEVEL", "info"),
			Format: getEnv("LOG_FORMAT", "json"),
		},
		Service: ServiceConfig{
			Name:        getEnv("SERVICE_NAME", "event-indexer"),
			Port:        getEnvAsInt("SERVICE_PORT", 8080),
			Environment: getEnv("ENVIRONMENT", "development"),
		},
	}

	// Load from YAML file if provided
	if yamlPath != "" {
		if err := loadFromYAML(yamlPath, config); err != nil {
			return nil, fmt.Errorf("failed to load config from YAML: %w", err)
		}
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return config, nil
}

// loadFromYAML loads configuration from a YAML file
func loadFromYAML(path string, config *Config) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	return yaml.Unmarshal(data, config)
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Database.URL == "" {
		return fmt.Errorf("database URL is required")
	}

	if c.RPC.Endpoint == "" {
		return fmt.Errorf("RPC endpoint is required")
	}

	if c.Indexer.BatchSize <= 0 {
		return fmt.Errorf("indexer batch size must be positive")
	}

	if c.Indexer.DefaultConfirmBlocks < 1 || c.Indexer.DefaultConfirmBlocks > 100 {
		return fmt.Errorf("default confirm blocks must be between 1 and 100")
	}

	if c.Service.Port <= 0 || c.Service.Port > 65535 {
		return fmt.Errorf("service port must be between 1 and 65535")
	}

	return nil
}

// Helper functions for environment variables

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

func getEnvAsSlice(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		return strings.Split(value, ",")
	}
	return defaultValue
}

// IsDevelopment checks if the environment is development
func (c *Config) IsDevelopment() bool {
	return c.Service.Environment == "development"
}

// IsProduction checks if the environment is production
func (c *Config) IsProduction() bool {
	return c.Service.Environment == "production"
}

