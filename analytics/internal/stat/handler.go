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
	// `r` is already mounted at `/analytics/api/stats` by the server router.
	// Register endpoints directly under that prefix.
	r.GET("/:code", h.GetTotalClicks)

	// Clicks by country
	r.GET("/:code/geo", h.GetGeo)

	// Top referrer domains
	r.GET("/:code/referrers", h.GetReferrers)
}

func (h *Handler) GetTotalClicks(c *gin.Context) {
	code := c.Param("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "code is required"})
		return
	}

	totalClicks, err := h.service.GetTotalClicks(c.Request.Context(), code)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": totalClicks,
	})
}

func (h *Handler) GetGeo(c *gin.Context) {
	code := c.Param("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "code is required"})
		return
	}

	geo, err := h.service.GetGeo(c.Request.Context(), code)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": geo,
	})
}

func (h *Handler) GetReferrers(c *gin.Context) {
	code := c.Param("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "code is required"})
		return
	}

	referrers, err := h.service.GetReferrers(c.Request.Context(), code)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": referrers,
	})
}
