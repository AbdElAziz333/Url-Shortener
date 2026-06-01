package server

import (
	"net/http"

	"aziz.dev/redirect/internal/middleware"
	"aziz.dev/redirect/internal/resolve"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func NewRouter(resolverHandler *resolve.Handler) *gin.Engine {
	router := gin.New()
	router.HandleMethodNotAllowed = true
	router.Use(
		gin.Recovery(),
		gin.Logger(),
	)
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	redirectGroup := router.Group("/redirect")
	redirectGroup.Use(middleware.Prometheus())
	redirectGroup.GET("/health", healthHandler)

	resolverHandler.RegisterRoutes(redirectGroup)

	return router
}

func healthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "up"})
}
