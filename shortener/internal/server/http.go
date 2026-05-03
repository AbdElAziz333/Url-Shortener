package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func NewRouter(
	linkHandler *link.Handler,
) *gin.Engine {
	router := gin.New()
	router.Use(gin.Recovery(), gin.Logger())

	router.GET("/health/live", healthLiveHandler)
	router.GET("/health/ready", healthReadyHandler)

	linkGroup := router.Group("/api/links")
	linkHandler.RegisterRoutes(linkGroup)

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
