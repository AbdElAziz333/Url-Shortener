package server

import (
	"net/http"

	"aziz.dev/gateway/internal/user"
	"github.com/gin-gonic/gin"
)

func NewRouter(
	userHandler *user.Handler,
) *gin.Engine {
	router := gin.New()
	router.Use(gin.Recovery(), gin.Logger())
	router.GET("/health/live", healthLiveHandler)
	router.GET("/health/ready", healthReadyHandler)

	userGroup := router.Group("/api/users")
	userHandler.RegisterRoutes(userGroup)

	return router;
}

func healthLiveHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "up",
	})
}

func healthReadyHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "up",
	})
}