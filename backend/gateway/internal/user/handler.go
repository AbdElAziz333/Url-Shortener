package user

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	authRoutes := r.Group("/auth")
	authRoutes.POST("/register", h.Register)
	authRoutes.POST("/login", h.Login)
	authRoutes.POST("/refresh", h.Refresh)
	authRoutes.POST("/logout", h.Logout)
}

func (h *Handler) Register(c *gin.Context) {
	log := logrus.WithField("path", c.Request.URL.Path)
	log.Info("Handling register request")

	var r RegisterRequest

	if err := c.ShouldBindJSON(&r); err != nil {
		log.WithError(err).Warn("Invalid request body")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.service.Register(c, &r)
	if err != nil {
		log.WithError(err).Warn("Registration failed")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	log.Info("Registration successful")
	c.JSON(http.StatusOK, gin.H{"message": "User registered successfully..."})
}

func (h *Handler) Login(c *gin.Context) {
	log := logrus.WithField("path", c.Request.URL.Path)
	log.Info("Handling login request")

	var r LoginRequest

	if err := c.ShouldBindJSON(&r); err != nil {
		log.WithError(err).Warn("Invalid request body")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	t, err := h.service.Login(c, &r)
	if err != nil {
		log.WithError(err).Warn("Login failed")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.Header("access_token", t.AccessToken)
	// c.Header("refresh_token", t.RefreshToken)
	// maxAge := int((time.Duration(config.RefreshExpiry) * time.Second) - time.Now())
	c.SetCookie("refresh_token", t.RefreshToken, 60*60*24*7, "/", "localhost", false, true)
	log.Info("Login successful")
	c.JSON(http.StatusOK, gin.H{"message": "Logged in successfully..."})
}

func (h *Handler) Refresh(c *gin.Context) {
	log := logrus.WithField("path", c.Request.URL.Path)
	log.Info("Handling refresh request")

	t, err := h.service.Refresh(c)
	if err != nil {
		log.WithError(err).Warn("Refresh failed")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.Header("access_token", t.AccessToken)
	// c.Header("refresh_token", t.RefreshToken)

	c.SetCookie("refresh_token", t.RefreshToken, 60*60*24*7, "/", "localhost", false, true)

	log.Info("Refresh successful")
	c.JSON(http.StatusOK, gin.H{"message": "Refreshed token successfully..."})
}

func (h *Handler) Logout(c *gin.Context) {
	log := logrus.WithField("path", c.Request.URL.Path)
	log.Info("Handling logout request")

	c.Header("access_token", "")
	c.Header("refresh_token", "")
	log.Info("Logout successful")
	c.JSON(http.StatusOK, gin.H{"message": "Logged out Successfully..."})
}