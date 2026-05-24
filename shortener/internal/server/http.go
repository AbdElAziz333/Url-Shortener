package server

import (
	"net/http"

	"aziz.dev/shortener/internal/link"
	"aziz.dev/shortener/internal/middleware"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func NewRouter(
	linkHandler *link.Handler,
) *gin.Engine {
	router := gin.New()
	router.Use(
		gin.Recovery(),
		gin.Logger(),
	)
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	shortenerGroup := router.Group("/shortener")
	shortenerGroup.Use(middleware.Prometheus())
	
	shortenerGroup.GET("/health", healthHandler)

	linkGroup := shortenerGroup.Group("/api")
	linkHandler.RegisterRoutes(linkGroup)

	return router;
}

func healthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "OK",
	})
}