package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/smart-contract-event-indexer/api-gateway/internal/config"
	"github.com/smart-contract-event-indexer/shared/utils"
)

type apiTier string

const (
	tierFree apiTier = "free"
	tierPro  apiTier = "pro"

	apiKeyContextKey  = "api_key"
	apiTierContextKey = "api_tier"
)

// APIKeyAuth enforces API key validation if keys are configured.
func APIKeyAuth(cfg *config.Config, logger utils.Logger) gin.HandlerFunc {
	freeKeys := sliceToSet(cfg.APIKeysFree)
	proKeys := sliceToSet(cfg.APIKeysPro)

	if len(freeKeys) == 0 && len(proKeys) == 0 {
		return func(c *gin.Context) {
			// Auth disabled for this environment
			c.Next()
		}
	}

	return func(c *gin.Context) {
		key := strings.TrimSpace(c.GetHeader("X-API-Key"))
		if key == "" {
			key = strings.TrimSpace(c.Query("api_key"))
		}

		if key == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing API key"})
			return
		}

		var tier apiTier
		switch {
		case proKeys[key]:
			tier = tierPro
		case freeKeys[key]:
			tier = tierFree
		default:
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid API key"})
			return
		}

		c.Set(apiKeyContextKey, key)
		c.Set(apiTierContextKey, tier)
		c.Next()
	}
}

// GetAPIKey retrieves the API key from context.
func GetAPIKey(c *gin.Context) string {
	if key, ok := c.Get(apiKeyContextKey); ok {
		if str, ok := key.(string); ok {
			return str
		}
	}
	return ""
}

// GetAPITier returns the caller tier.
func GetAPITier(c *gin.Context) apiTier {
	if tier, ok := c.Get(apiTierContextKey); ok {
		if t, ok := tier.(apiTier); ok {
			return t
		}
	}
	return tierFree
}

func sliceToSet(items []string) map[string]bool {
	set := make(map[string]bool, len(items))
	for _, item := range items {
		if trimmed := strings.TrimSpace(item); trimmed != "" {
			set[trimmed] = true
		}
	}
	return set
}
