package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func NewRouter(
	statHandler *stat.Handler,
) *gin.Engine {
	router := gin.New()
	router.Use(gin.Recovery(), gin.Logger())

	router.GET("/health/live", healthLiveHandler)
	router.GET("/health/ready", healthReadyHandler)

	statGroup := router.Group("/api/stats")
	statHandler.RegisterRoutes(statGroup)

	return router;
}

func healthLiveHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "OK",
	})
}

func healthReadyHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "OK",
	})
}
