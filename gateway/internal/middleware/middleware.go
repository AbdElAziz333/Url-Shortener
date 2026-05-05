package middleware

import (
	"net/http"

	"aziz.dev/gateway/internal/auth"
	"aziz.dev/gateway/internal/config"
	"github.com/gin-gonic/gin"
)

func AccessTokenMiddleware(cfg *config.JWTConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("Authorization")
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			c.Abort()
			return
		}

		if !auth.ValidateAccessToken(cfg, token) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			c.Abort()
			return
		}
		c.Next()
	}
}