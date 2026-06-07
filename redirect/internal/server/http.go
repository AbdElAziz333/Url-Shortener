package server

import (
	"net/http"

	"aziz.dev/redirect/internal/resolve"
	"github.com/gin-gonic/gin"
)

func NewRouter(resolverHandler *resolve.Handler) *gin.Engine {
	router := gin.New()
	router.HandleMethodNotAllowed = true
	router.Use(
		gin.Recovery(),
		gin.Logger(),
	)

	redirectGroup := router.Group("/redirect")
	redirectGroup.GET("/health", healthHandler)

	resolverHandler.RegisterRoutes(redirectGroup)

	return router
}

func healthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "up"})
}
