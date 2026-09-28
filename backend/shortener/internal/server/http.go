package server

import (
	"net/http"

	"aziz.dev/shortener/internal/link"
	"github.com/gin-gonic/gin"
)

func NewRouter(
	linkHandler *link.Handler,
) *gin.Engine {
	router := gin.New()
	router.Use(
		gin.Recovery(),
		gin.Logger(),
	)

	shortenerGroup := router.Group("/shortener")
	shortenerGroup.GET("/health", healthHandler)

	linkGroup := shortenerGroup.Group("/api")
	linkHandler.RegisterRoutes(linkGroup)

	return router
}

func healthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "OK",
	})
}
