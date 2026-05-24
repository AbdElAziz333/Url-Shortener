package server

import (
	"net/http"

	"aziz.dev/analytics/internal/middleware"
	"aziz.dev/analytics/internal/stat"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func NewRouter(
	statHandler *stat.Handler,
) *gin.Engine {
	router := gin.New()
	router.Use(
		gin.Recovery(),
		gin.Logger(),
	)

	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	analyticsGroup := router.Group("/analytics")
	analyticsGroup.Use(middleware.Prometheus())
	analyticsGroup.GET("/health", healthHandler)

	statGroup := analyticsGroup.Group("/api/stats")
	statHandler.RegisterRoutes(statGroup)

	return router;
}

func healthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "OK",
	})
}