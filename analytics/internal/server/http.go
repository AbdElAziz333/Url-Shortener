package server

import (
	"net/http"

	"aziz.dev/analytics/internal/stat"
	"github.com/gin-gonic/gin"
)

func NewRouter(
	statHandler *stat.Handler,
) *gin.Engine {
	router := gin.New()
	router.Use(gin.Recovery(), gin.Logger())
	analyticsGroup := router.Group("/analytics")

	analyticsGroup.GET("/health", healthHandler)

	statGroup := router.Group("/api/stats")
	statHandler.RegisterRoutes(statGroup)

	return router;
}

func healthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "OK",
	})
}