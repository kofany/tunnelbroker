package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kofany/tunnelbroker/internal/config"
)

// APIKeyAuth checks if the request contains a valid API key
func APIKeyAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := c.GetHeader("X-API-Key")
		if apiKey == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Missing API key"})
			return
		}

		if apiKey != config.GlobalConfig.API.Key {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid API key"})
			return
		}

		c.Next()
	}
}
