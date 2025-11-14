package middleware

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// RateLimiter enforces a simple fixed-window rate limit using Redis.
func RateLimiter(redisClient *redis.Client, limit int, window time.Duration) gin.HandlerFunc {
	if limit <= 0 {
		limit = 100
	}

	return func(c *gin.Context) {
		ctx := c.Request.Context()
		key := fmt.Sprintf("rate:%s:%s", c.ClientIP(), c.FullPath())

		count, err := redisClient.Incr(ctx, key).Result()
		if err != nil {
			c.Next()
			return
		}

		if count == 1 {
			redisClient.Expire(ctx, key, window)
		}

		if count > int64(limit) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "rate limit exceeded",
			})
			return
		}

		c.Next()
	}
}
