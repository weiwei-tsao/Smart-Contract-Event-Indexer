package database

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/smart-contract-event-indexer/shared/config"
	"github.com/smart-contract-event-indexer/shared/utils"
)

// RedisClient wraps redis.Client with additional functionality
type RedisClient struct {
	*redis.Client
	logger utils.Logger
}

// NewRedisClient creates a new Redis client
func NewRedisClient(cfg config.RedisConfig, logger utils.Logger) (*RedisClient, error) {
	opt, err := redis.ParseURL(cfg.URL)
	if err != nil {
		return nil, utils.WrapError(utils.ErrCodeRedisConnection, "failed to parse Redis URL", err)
	}

	if cfg.Password != "" {
		opt.Password = cfg.Password
	}
	opt.DB = cfg.DB

	client := redis.NewClient(opt)

	// Verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, utils.WrapError(utils.ErrCodeRedisConnection, "failed to ping Redis", err)
	}

	logger.Info("Redis connection established")

	return &RedisClient{
		Client: client,
		logger: logger,
	}, nil
}

// HealthCheck performs a Redis health check
func (r *RedisClient) HealthCheck(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	if err := r.Ping(ctx).Err(); err != nil {
		return utils.WrapError(utils.ErrCodeRedisConnection, "Redis health check failed", err)
	}

	return nil
}

// Close closes the Redis connection
func (r *RedisClient) Close() error {
	r.logger.Info("Closing Redis connection")
	return r.Client.Close()
}

