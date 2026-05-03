package user

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	authRoutes := r.Group("/auth")
	authRoutes.GET("/register", h.Register)
	authRoutes.GET("/login", h.Login)
	authRoutes.GET("/refresh", h.Refresh)
	authRoutes.GET("/logout", h.Logout)
}

func (h *Handler) Register(c *gin.Context) {
	var r *RegisterRequest

	if err := c.ShouldBindJSON(&r); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	}

	err := h.service.Register(c, r)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User registered successfully..."})
}

func (h *Handler) Login(c *gin.Context) {
	var r *LoginRequest

	if err := c.ShouldBindJSON(r); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.service.Login(c, r)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Logged in successfully..."})
}

func (h *Handler) Refresh(c *gin.Context) {
	h.service.Refresh(c)
	c.JSON(http.StatusOK, gin.H{"message": "Refreshed token successfully..."})
}

func (h *Handler) Logout(c *gin.Context) {
	c.Header("access_token", "")
	c.Header("refresh_token", "")
	c.JSON(http.StatusOK, gin.H{"message": "Logged out Successfully..."})
}