package middleware

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// RateLimiter enforces a simple fixed-window rate limit using Redis.
func RateLimiter(redisClient *redis.Client, freeLimit, proLimit int, window time.Duration) gin.HandlerFunc {
	if freeLimit <= 0 {
		freeLimit = 100
	}
	if proLimit <= 0 {
		proLimit = freeLimit * 10
	}

	return func(c *gin.Context) {
		ctx := c.Request.Context()
		keyID := c.ClientIP()
		if apiKey := GetAPIKey(c); apiKey != "" {
			keyID = apiKey
		}

		tier := GetAPITier(c)
		permitted := freeLimit
		if tier == tierPro {
			permitted = proLimit
		}

		cacheKey := fmt.Sprintf("rate:%s:%s:%s", tier, keyID, c.FullPath())

		count, err := redisClient.Incr(ctx, cacheKey).Result()
		if err != nil {
			c.Next()
			return
		}

		if count == 1 {
			redisClient.Expire(ctx, cacheKey, window)
		}

		if count > int64(permitted) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "rate limit exceeded",
			})
			return
		}

		c.Next()
	}
}
