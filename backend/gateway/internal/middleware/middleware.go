package middleware

import (
	"net/http"
	"strings"

	"aziz.dev/gateway/internal/auth"
	"aziz.dev/gateway/internal/config"
	"github.com/gin-gonic/gin"
)

func AccessTokenMiddleware(cfg *config.JWTConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := strings.TrimSpace(c.GetHeader("Authorization"))
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			c.Abort()
			return
		}

		// Allow both "Bearer <token>" and raw token values.
		if len(token) > 7 && strings.EqualFold(token[:7], "Bearer ") {
			token = strings.TrimSpace(token[7:])
		}

		claims, valid := auth.ValidateAccessToken(cfg, token)
		if !valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized validating the token"})
			c.Abort()
			return
		}
		
		c.Set("userID", claims.UserID.String())
		c.Next()
	}
}