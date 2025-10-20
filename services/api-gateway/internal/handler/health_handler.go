package handler

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/smart-contract-event-indexer/shared/utils"
)

// HealthHandler handles health check requests
type HealthHandler struct {
	db          *sql.DB
	redisClient *redis.Client
	logger      utils.Logger
}

// NewHealthHandler creates a new HealthHandler
func NewHealthHandler(
	db *sql.DB,
	redisClient *redis.Client,
	logger utils.Logger,
) *HealthHandler {
	return &HealthHandler{
		db:          db,
		redisClient: redisClient,
		logger:      logger,
	}
}

// HealthCheck handles GET /api/v1/health
func (h *HealthHandler) HealthCheck(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	status := "healthy"
	services := make(map[string]interface{})

	// Check database
	dbStart := time.Now()
	if err := h.db.PingContext(ctx); err != nil {
		status = "unhealthy"
		services["database"] = map[string]interface{}{
			"status":  "error",
			"error":   err.Error(),
			"latency": time.Since(dbStart).Milliseconds(),
		}
	} else {
		services["database"] = map[string]interface{}{
			"status":  "healthy",
			"latency": time.Since(dbStart).Milliseconds(),
		}
	}

	// Check Redis
	redisStart := time.Now()
	if err := h.redisClient.Ping(ctx).Err(); err != nil {
		status = "unhealthy"
		services["redis"] = map[string]interface{}{
			"status":  "error",
			"error":   err.Error(),
			"latency": time.Since(redisStart).Milliseconds(),
		}
	} else {
		services["redis"] = map[string]interface{}{
			"status":  "healthy",
			"latency": time.Since(redisStart).Milliseconds(),
		}
	}

	// Set HTTP status code
	httpStatus := http.StatusOK
	if status == "unhealthy" {
		httpStatus = http.StatusServiceUnavailable
	}

	c.JSON(httpStatus, gin.H{
		"status":    status,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"services":  services,
	})
}
