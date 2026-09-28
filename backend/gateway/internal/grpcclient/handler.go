package grpcclient

import (
	"net/http"
	"time"

	redirectpb "aziz.dev/gateway/internal/pb/redirect"
	shortenerpb "aziz.dev/gateway/internal/pb/shortener"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Handler struct {
	clients *Clients
}

func NewHandler(clients *Clients) *Handler {
	return &Handler{clients: clients}
}

func (h *Handler) GetAllLinks(c *gin.Context) {
	userID, ok := userIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or missing user id"})
		return
	}

	resp, err := h.clients.Shortener.GetAllLinks(c.Request.Context(), &shortenerpb.GetAllLinksRequest{
		UserId: userID.String(),
	})
	if err != nil {
		writeGRPCError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": protoLinksToJSON(resp.GetLinks())})
}

func (h *Handler) CreateLink(c *gin.Context) {
	userID, ok := userIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or missing user id"})
		return
	}

	var body struct {
		OriginalURL string     `json:"original_url" binding:"required,url"`
		CustomAlias string     `json:"custom_alias,omitempty" binding:"omitempty,alphanum"`
		ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	req := &shortenerpb.CreateLinkRequest{
		UserId:      userID.String(),
		OriginalUrl: body.OriginalURL,
	}
	if body.CustomAlias != "" {
		req.CustomAlias = &body.CustomAlias
	}
	if body.ExpiresAt != nil {
		req.ExpiresAt = timestamppb.New(*body.ExpiresAt)
	}

	resp, err := h.clients.Shortener.CreateLink(c.Request.Context(), req)
	if err != nil {
		writeGRPCError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": protoLinkToJSON(resp.GetLink())})
}

func (h *Handler) UpdateExpiry(c *gin.Context) {
	userID, ok := userIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or missing user id"})
		return
	}

	var body struct {
		ExpiresAt time.Time `json:"expires_at" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err := h.clients.Shortener.UpdateExpiry(c.Request.Context(), &shortenerpb.UpdateExpiryRequest{
		UserId:    userID.String(),
		Code:      c.Param("code"),
		ExpiresAt: timestamppb.New(body.ExpiresAt),
	})
	if err != nil {
		writeGRPCError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "successfully updated expiry"})
}

func (h *Handler) DeleteLink(c *gin.Context) {
	userID, ok := userIDFromContext(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or missing user id"})
		return
	}

	_, err := h.clients.Shortener.DeleteLink(c.Request.Context(), &shortenerpb.DeleteLinkRequest{
		UserId: userID.String(),
		Code:   c.Param("code"),
	})
	if err != nil {
		writeGRPCError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "successfully deleted link"})
}

func (h *Handler) ResolveCode(c *gin.Context) {
	code := c.Param("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "code is required"})
		return
	}

	resp, err := h.clients.Redirect.ResolveCode(c.Request.Context(), &redirectpb.ResolveCodeRequest{
		Code: code,
	})
	if err != nil {
		writeGRPCError(c, err)
		return
	}

	c.Redirect(http.StatusFound, resp.GetOriginalUrl())
}

func userIDFromContext(c *gin.Context) (uuid.UUID, bool) {
	v, ok := c.Get("userID")
	if !ok {
		return uuid.UUID{}, false
	}
	id, ok := v.(string)
	if !ok || id == "" {
		return uuid.UUID{}, false
	}
	userID, err := uuid.Parse(id)
	if err != nil {
		return uuid.UUID{}, false
	}
	return userID, true
}

func writeGRPCError(c *gin.Context, err error) {
	st, ok := status.FromError(err)
	if !ok {
		logrus.WithError(err).Error("unexpected gRPC error")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	switch st.Code() {
	case codes.InvalidArgument:
		c.JSON(http.StatusBadRequest, gin.H{"error": st.Message()})
	case codes.NotFound:
		c.JSON(http.StatusNotFound, gin.H{"error": st.Message()})
	case codes.FailedPrecondition:
		c.JSON(http.StatusGone, gin.H{"error": st.Message()})
	case codes.Unavailable:
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": st.Message()})
	default:
		logrus.WithError(err).Error("gRPC call failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": st.Message()})
	}
}

func protoLinksToJSON(links []*shortenerpb.Link) []gin.H {
	out := make([]gin.H, 0, len(links))
	for _, link := range links {
		out = append(out, protoLinkToJSON(link))
	}
	return out
}

func protoLinkToJSON(link *shortenerpb.Link) gin.H {
	if link == nil {
		return gin.H{}
	}

	item := gin.H{
		"code":         link.GetCode(),
		"original_url": link.GetOriginalUrl(),
		"is_active":    link.GetIsActive(),
	}
	if link.CustomAlias != nil {
		item["custom_alias"] = link.GetCustomAlias()
	}
	if link.ExpiresAt != nil {
		item["expires_at"] = link.GetExpiresAt().AsTime()
	}
	if link.CreatedAt != nil {
		item["created_at"] = link.GetCreatedAt().AsTime()
	}
	return item
}
