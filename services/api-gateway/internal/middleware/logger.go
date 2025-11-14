package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/smart-contract-event-indexer/shared/utils"
)

// Logger returns a gin.HandlerFunc for logging requests
func Logger(logger utils.Logger) gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		logger.Info("HTTP Request",
			"method", param.Method,
			"path", param.Path,
			"status", param.StatusCode,
			"latency", param.Latency,
			"client_ip", param.ClientIP,
			"user_agent", param.Request.UserAgent(),
		)
		return ""
	})
}

// Recovery returns a gin.HandlerFunc for recovering from panics
func Recovery(logger utils.Logger) gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		logger.Error("Panic recovered",
			"error", recovered,
			"path", c.Request.URL.Path,
		)
		c.AbortWithStatus(500)
	})
}
