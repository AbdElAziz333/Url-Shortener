package stat

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
	linkRoutes := r.Group("/stats")
	linkRoutes.GET("/:code", h.GetTotalClicks)
	linkRoutes.GET("/:code/geo", h.GetGeo)
	linkRoutes.GET("/:code/referrers", h.GetReferrers)
}

func (h *Handler) GetTotalClicks(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "OK",
	})
}

func (h *Handler) GetGeo(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "OK",
	})
}

func (h *Handler) GetReferrers(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "OK",
	})
}