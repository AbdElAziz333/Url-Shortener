package resolve

import (
	"errors"
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
	r.GET("/:code", h.ResolveCode)
}

func (h *Handler) ResolveCode(c *gin.Context) {
	code := c.Param("code")
	logrus.WithField("code", code).Info("Received resolve request")

	if code == "" {
		logrus.Warn("Missing code parameter")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "code is required",
		})
		return
	}

	redirect, err := h.service.ResolveCode(c.Request.Context(), code)
	if err != nil {
		if errors.Is(err, ErrLinkInactive) {
			logrus.WithField("code", code).Warn("Link is inactive or expired")
			c.JSON(http.StatusGone, gin.H{"error": err.Error()})
			return			
		}
		
		logrus.WithError(err).WithField("code", code).Error("Failed to resolve code")
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	logrus.WithFields(logrus.Fields{
		"code":         code,
		"original_url": redirect.OriginalURL,
	}).Info("Successfully resolved code, redirecting")
	c.Redirect(http.StatusFound, redirect.OriginalURL)
}