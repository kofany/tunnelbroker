package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kofany/tunnelbroker/internal/config"
)

// APIKeyAuth sprawdza czy żądanie zawiera prawidłowy klucz API
func APIKeyAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := c.GetHeader("X-API-Key")
		if apiKey == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Brak klucza API"})
			return
		}

		if apiKey != config.GlobalConfig.API.Key {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Nieprawidłowy klucz API"})
			return
		}

		c.Next()
	}
} 