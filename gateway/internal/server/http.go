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
	gatewayGroup := router.Group("/gateway")

	gatewayGroup.GET("/health", healthHandler)

	userGroup := gatewayGroup.Group("/")
	userHandler.RegisterRoutes(userGroup)

	return router;
}

func healthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "up",
	})
}