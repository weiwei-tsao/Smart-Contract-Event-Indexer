package cache

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/smart-contract-event-indexer/query-service/internal/config"
	"go.uber.org/zap"
)

// CacheManager handles caching operations
type CacheManager struct {
	client    *redis.Client
	logger    *zap.Logger
	defaultTTL time.Duration
}

// NewCacheManager creates a new cache manager
func NewCacheManager(client *redis.Client, logger *zap.Logger, defaultTTL time.Duration) *CacheManager {
	return &CacheManager{
		client:     client,
		logger:     logger,
		defaultTTL: defaultTTL,
	}
}

// CacheKey represents a cache key with metadata
type CacheKey struct {
	Type    string
	Hash    string
	Version string
}

// NewCacheKey creates a new cache key
func NewCacheKey(cacheType, hash, version string) *CacheKey {
	return &CacheKey{
		Type:    cacheType,
		Hash:    hash,
		Version: version,
	}
}

// String returns the cache key as a string
func (k *CacheKey) String() string {
	return fmt.Sprintf("%s:%s:%s", k.Type, k.Hash, k.Version)
}

// GenerateHash generates a hash for the given data
func GenerateHash(data interface{}) (string, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return "", err
	}

	hash := sha256.Sum256(jsonData)
	return fmt.Sprintf("%x", hash), nil
}

// Get retrieves a value from cache
func (c *CacheManager) Get(ctx context.Context, key *CacheKey, dest interface{}) error {
	keyStr := key.String()
	
	val, err := c.client.Get(ctx, keyStr).Result()
	if err != nil {
		if err == redis.Nil {
			c.logger.Debug("Cache miss", zap.String("key", keyStr))
			return ErrCacheMiss
		}
		c.logger.Error("Cache get error", zap.String("key", keyStr), zap.Error(err))
		return err
	}

	if err := json.Unmarshal([]byte(val), dest); err != nil {
		c.logger.Error("Cache unmarshal error", zap.String("key", keyStr), zap.Error(err))
		return err
	}

	c.logger.Debug("Cache hit", zap.String("key", keyStr))
	return nil
}

// Set stores a value in cache
func (c *CacheManager) Set(ctx context.Context, key *CacheKey, value interface{}, ttl time.Duration) error {
	keyStr := key.String()
	
	jsonData, err := json.Marshal(value)
	if err != nil {
		c.logger.Error("Cache marshal error", zap.String("key", keyStr), zap.Error(err))
		return err
	}

	if ttl == 0 {
		ttl = c.defaultTTL
	}

	if err := c.client.Set(ctx, keyStr, jsonData, ttl).Err(); err != nil {
		c.logger.Error("Cache set error", zap.String("key", keyStr), zap.Error(err))
		return err
	}

	c.logger.Debug("Cache set", zap.String("key", keyStr), zap.Duration("ttl", ttl))
	return nil
}

// Delete removes a value from cache
func (c *CacheManager) Delete(ctx context.Context, key *CacheKey) error {
	keyStr := key.String()
	
	if err := c.client.Del(ctx, keyStr).Err(); err != nil {
		c.logger.Error("Cache delete error", zap.String("key", keyStr), zap.Error(err))
		return err
	}

	c.logger.Debug("Cache delete", zap.String("key", keyStr))
	return nil
}

// DeletePattern removes all keys matching a pattern
func (c *CacheManager) DeletePattern(ctx context.Context, pattern string) error {
	keys, err := c.client.Keys(ctx, pattern).Result()
	if err != nil {
		c.logger.Error("Cache keys error", zap.String("pattern", pattern), zap.Error(err))
		return err
	}

	if len(keys) == 0 {
		return nil
	}

	if err := c.client.Del(ctx, keys...).Err(); err != nil {
		c.logger.Error("Cache delete pattern error", zap.String("pattern", pattern), zap.Error(err))
		return err
	}

	c.logger.Debug("Cache delete pattern", zap.String("pattern", pattern), zap.Int("count", len(keys)))
	return nil
}

// InvalidateContractCache invalidates cache for a specific contract
func (c *CacheManager) InvalidateContractCache(ctx context.Context, contractAddress string) error {
	patterns := []string{
		fmt.Sprintf("query:*contract:%s*", contractAddress),
		fmt.Sprintf("stats:*contract:%s*", contractAddress),
		fmt.Sprintf("events:*contract:%s*", contractAddress),
	}

	for _, pattern := range patterns {
		if err := c.DeletePattern(ctx, pattern); err != nil {
			return err
		}
	}

	c.logger.Info("Invalidated contract cache", zap.String("contract", contractAddress))
	return nil
}

// InvalidateAllCache invalidates all cache
func (c *CacheManager) InvalidateAllCache(ctx context.Context) error {
	patterns := []string{
		"query:*",
		"stats:*",
		"events:*",
	}

	for _, pattern := range patterns {
		if err := c.DeletePattern(ctx, pattern); err != nil {
			return err
		}
	}

	c.logger.Info("Invalidated all cache")
	return nil
}

// GetStats returns cache statistics
func (c *CacheManager) GetStats(ctx context.Context) (*CacheStats, error) {
	info, err := c.client.Info(ctx, "stats").Result()
	if err != nil {
		return nil, err
	}

	// Parse Redis info to extract cache statistics
	// This is a simplified version - in production you'd want more detailed parsing
	stats := &CacheStats{
		HitRate: 0.0, // Would need to calculate from Redis info
		Keys:    0,   // Would need to get from DBSIZE
		Memory:  0,   // Would need to get from memory usage
	}

	return stats, nil
}

// CacheStats represents cache statistics
type CacheStats struct {
	HitRate float64 `json:"hit_rate"`
	Keys    int64   `json:"keys"`
	Memory  int64   `json:"memory_bytes"`
}

// Cache errors
var (
	ErrCacheMiss = fmt.Errorf("cache miss")
)
