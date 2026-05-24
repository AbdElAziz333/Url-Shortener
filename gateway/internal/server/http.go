package server

import (
	"net/http"

	"aziz.dev/gateway/internal/middleware"
	"aziz.dev/gateway/internal/user"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func NewRouter(
	userHandler *user.Handler,
) *gin.Engine {
	router := gin.New()
	router.Use(
		gin.Recovery(),
		gin.Logger(),
	)

	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	gatewayGroup := router.Group("/gateway")
	gatewayGroup.Use(middleware.Prometheus())
	
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