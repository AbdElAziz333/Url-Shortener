package link

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{
		service: service,
	}
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	linkRoutes := r.Group("/links")
	linkRoutes.GET("/", h.GetAll)
	linkRoutes.POST("/", h.Create)
	linkRoutes.PATCH("/:code/expiry", h.UpdateExpiry)
	linkRoutes.DELETE("/:code", h.Delete)
}

func (h *Handler) GetAll(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "OK",
	})
}

func (h *Handler) Create(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "OK",
	})
}

func (h *Handler) UpdateExpiry(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "OK",
	})
}

func (h *Handler) Delete(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "OK",
	})
}
