package link

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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
	linkRoutes.GET("", h.GetAll)
	linkRoutes.POST("", h.Create)
	linkRoutes.PATCH("/:code/expiry", h.UpdateExpiry)
	linkRoutes.DELETE("/:code", h.Delete)
}

func (h *Handler) GetAll(c *gin.Context) {
	userID, err := uuid.Parse(c.GetHeader("User-ID"))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or missing user id"})
		return
	}

	links, err := h.service.GetAll(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})

		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":    links,
	})
}

func (h *Handler) Create(c *gin.Context) {
	userID, err := uuid.Parse(c.GetHeader("User-ID"))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or missing user id"})
		return
	}

	var r *CreateRequest
	if err := c.ShouldBindJSON(&r); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	d, err := h.service.Create(c.Request.Context(), userID, *r)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})

		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":    d,
	})
}

func (h *Handler) UpdateExpiry(c *gin.Context) {
	userID, err := uuid.Parse(c.GetHeader("User-ID"))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or missing user id"})
		return
	}
	var code = c.Param("code")

	var r *UpdateExpiryDto
	if err := c.ShouldBindJSON(&r); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	if err := h.service.UpdateExpiry(c.Request.Context(), userID, code, *r); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})

		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "successfully updated expiry",
	})
}

func (h *Handler) Delete(c *gin.Context) {
	userID, err := uuid.Parse(c.GetHeader("User-ID"))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or missing user id"})
		return
	}
	var code = c.Param("code")

	if err := h.service.Delete(c.Request.Context(), userID, code); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})

		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "successfully deleted link",
	})
}
