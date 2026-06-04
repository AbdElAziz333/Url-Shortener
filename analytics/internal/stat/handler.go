package stat

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
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
		logrus.Warn("Request to GetTotalClicks without code parameter")
		c.JSON(http.StatusBadRequest, gin.H{"error": "code is required"})
		return
	}

	logrus.WithField("code", code).Debug("Handling GetTotalClicks request")
	totalClicks, err := h.service.GetTotalClicks(c.Request.Context(), code)
	if err != nil {
		logrus.WithError(err).WithField("code", code).Error("Failed to get total clicks")
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
		logrus.Warn("Request to GetGeo without code parameter")
		c.JSON(http.StatusBadRequest, gin.H{"error": "code is required"})
		return
	}

	logrus.WithField("code", code).Debug("Handling GetGeo request")
	geo, err := h.service.GetGeo(c.Request.Context(), code)
	if err != nil {
		logrus.WithError(err).WithField("code", code).Error("Failed to get geo stats")
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
		logrus.Warn("Request to GetReferrers without code parameter")
		c.JSON(http.StatusBadRequest, gin.H{"error": "code is required"})
		return
	}

	logrus.WithField("code", code).Debug("Handling GetReferrers request")
	referrers, err := h.service.GetReferrers(c.Request.Context(), code)
	if err != nil {
		logrus.WithError(err).WithField("code", code).Error("Failed to get referrer stats")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": referrers,
	})
}
