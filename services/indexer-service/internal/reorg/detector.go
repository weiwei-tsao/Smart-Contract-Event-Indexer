package reorg

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/smart-contract-event-indexer/shared/models"
	"github.com/smart-contract-event-indexer/shared/utils"
)

const (
	// BlockCacheSize is the number of recent blocks to cache
	BlockCacheSize = 50
	// BlockCacheKeyPrefix is the Redis key prefix for block cache
	BlockCacheKeyPrefix = "block:hash:"
)

// Detector detects blockchain reorganizations
type Detector struct {
	redisClient *redis.Client
	logger      utils.Logger
	cacheSize   int
}

// NewDetector creates a new reorg detector
func NewDetector(redisClient *redis.Client, logger utils.Logger) *Detector {
	return &Detector{
		redisClient: redisClient,
		logger:      logger,
		cacheSize:   BlockCacheSize,
	}
}

// BlockInfo represents information about a block
type BlockInfo struct {
	Number     int64
	Hash       models.Hash
	ParentHash models.Hash
	Timestamp  time.Time
}

// CacheBlock stores a block's hash in the cache
func (d *Detector) CacheBlock(ctx context.Context, block *BlockInfo) error {
	key := fmt.Sprintf("%s%d", BlockCacheKeyPrefix, block.Number)
	
	// Store block hash with expiration (keep for 7 days)
	err := d.redisClient.Set(ctx, key, string(block.Hash), 7*24*time.Hour).Err()
	if err != nil {
		return fmt.Errorf("failed to cache block: %w", err)
	}
	
	d.logger.WithFields(map[string]interface{}{
		"block_number": block.Number,
		"block_hash":   block.Hash,
	}).Debug("Block cached")
	
	return nil
}

// GetCachedBlockHash retrieves a block's hash from the cache
func (d *Detector) GetCachedBlockHash(ctx context.Context, blockNumber int64) (models.Hash, error) {
	key := fmt.Sprintf("%s%d", BlockCacheKeyPrefix, blockNumber)
	
	hash, err := d.redisClient.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return "", fmt.Errorf("block %d not in cache", blockNumber)
		}
		return "", fmt.Errorf("failed to get cached block: %w", err)
	}
	
	return models.Hash(hash), nil
}

// DetectReorg checks if a reorganization has occurred
func (d *Detector) DetectReorg(ctx context.Context, currentBlock *BlockInfo) (bool, int64, error) {
	// Get the cached hash for the parent block
	cachedParentHash, err := d.GetCachedBlockHash(ctx, currentBlock.Number-1)
	if err != nil {
		// If parent block is not cached, assume no reorg (could be initial sync)
		d.logger.WithFields(map[string]interface{}{
			"block_number":     currentBlock.Number,
			"parent_block":     currentBlock.Number - 1,
		}).Debug("Parent block not in cache, assuming no reorg")
		
		// Cache the current block
		d.CacheBlock(ctx, currentBlock)
		return false, 0, nil
	}
	
	// Compare parent hashes
	if cachedParentHash != currentBlock.ParentHash {
		// Reorg detected! Find the fork point
		d.logger.WithFields(map[string]interface{}{
			"block_number":        currentBlock.Number,
			"expected_parent":     cachedParentHash,
			"actual_parent":       currentBlock.ParentHash,
		}).Warn("Blockchain reorganization detected")
		
		forkPoint, err := d.findForkPoint(ctx, currentBlock)
		if err != nil {
			return true, 0, fmt.Errorf("failed to find fork point: %w", err)
		}
		
		d.logger.WithField("fork_point", forkPoint).Info("Fork point identified")
		
		return true, forkPoint, nil
	}
	
	// No reorg, cache the current block
	d.CacheBlock(ctx, currentBlock)
	
	return false, 0, nil
}

// findForkPoint finds the block number where the chains diverged
func (d *Detector) findForkPoint(ctx context.Context, currentBlock *BlockInfo) (int64, error) {
	// Start from the parent block and go backwards
	blockNumber := currentBlock.Number - 1
	
	// We'll check up to cacheSize blocks back
	for i := 0; i < d.cacheSize && blockNumber > 0; i++ {
		cachedHash, err := d.GetCachedBlockHash(ctx, blockNumber)
		if err != nil {
			// If we can't find the cached block, assume this is the fork point
			d.logger.WithField("block_number", blockNumber).Debug("Cached block not found, assuming fork point")
			return blockNumber + 1, nil
		}
		
		// In a real implementation, we'd fetch the actual block from the chain
		// and compare hashes. Since this is a simplified detector, we treat the
		// first cached hash we encounter as the fork point so we at least have
		// a deterministic rollback target.
		d.logger.WithFields(map[string]interface{}{
			"block_number": blockNumber,
			"cached_hash":  cachedHash,
		}).Debug("Using cached block as fork point")
		return blockNumber, nil
	}
	
	// If we've gone back cacheSize blocks and haven't found a match,
	// the reorg is deeper than our cache. Return the oldest cached block
	d.logger.Warn("Reorg depth exceeds cache size")
	return currentBlock.Number - int64(d.cacheSize), nil
}

// ClearCache clears all cached block hashes
func (d *Detector) ClearCache(ctx context.Context) error {
	// Find all block cache keys
	iter := d.redisClient.Scan(ctx, 0, BlockCacheKeyPrefix+"*", 0).Iterator()
	
	count := 0
	for iter.Next(ctx) {
		if err := d.redisClient.Del(ctx, iter.Val()).Err(); err != nil {
			d.logger.WithError(err).Warn("Failed to delete cache key")
		}
		count++
	}
	
	if err := iter.Err(); err != nil {
		return fmt.Errorf("failed to iterate cache keys: %w", err)
	}
	
	d.logger.WithField("deleted_keys", count).Info("Block cache cleared")
	
	return nil
}

// GetCacheStats returns statistics about the block cache
func (d *Detector) GetCacheStats(ctx context.Context) (map[string]interface{}, error) {
	// Count cached blocks
	iter := d.redisClient.Scan(ctx, 0, BlockCacheKeyPrefix+"*", 0).Iterator()
	
	count := 0
	for iter.Next(ctx) {
		count++
	}
	
	if err := iter.Err(); err != nil {
		return nil, fmt.Errorf("failed to count cache keys: %w", err)
	}
	
	stats := map[string]interface{}{
		"cached_blocks": count,
		"cache_size":    d.cacheSize,
	}
	
	return stats, nil
}

// InitializeCache initializes the cache with recent blocks
func (d *Detector) InitializeCache(ctx context.Context, blocks []*BlockInfo) error {
	d.logger.WithField("block_count", len(blocks)).Info("Initializing block cache")
	
	for _, block := range blocks {
		if err := d.CacheBlock(ctx, block); err != nil {
			return fmt.Errorf("failed to cache block %d: %w", block.Number, err)
		}
	}
	
	return nil
}
