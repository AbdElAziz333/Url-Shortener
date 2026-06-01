package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"aziz.dev/shortener/internal/link"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newRouter builds the router with a real (but service-less) link.Handler.
// Handler-level behaviour is covered in handler_test.go; here we only care
// about routing, middleware wiring, and the health endpoint.
func newRouter(t *testing.T) *gin.Engine {
	t.Helper()
	// NewHandler accepts a Service interface — pass nil; none of the routes
	// exercised below invoke the service.
	h := link.NewHandler(nil)
	return NewRouter(h)
}

// --- Health endpoint ---

func TestRouter_HealthEndpoint_Returns200(t *testing.T) {
	r := newRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/shortener/health", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRouter_HealthEndpoint_ReturnsOKMessage(t *testing.T) {
	r := newRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/shortener/health", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "OK", body["message"])
}

func TestRouter_HealthEndpoint_ContentTypeIsJSON(t *testing.T) {
	r := newRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/shortener/health", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")
}

// --- Metrics endpoint ---

func TestRouter_MetricsEndpoint_Returns200(t *testing.T) {
	r := newRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

// --- Route registration: link routes are mounted under /shortener/api/links ---

func TestRouter_LinkRoutes_GetAllRequiresUserIDHeader(t *testing.T) {
	r := newRouter(t)
	// No User-ID header → handler returns 401 before touching the service
	req := httptest.NewRequest(http.MethodGet, "/shortener/api/links", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// 401 confirms the route exists and the handler is wired correctly
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestRouter_LinkRoutes_PostRequiresUserIDHeader(t *testing.T) {
	r := newRouter(t)
	req := httptest.NewRequest(http.MethodPost, "/shortener/api/links", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestRouter_LinkRoutes_PatchExpiryRequiresUserIDHeader(t *testing.T) {
	r := newRouter(t)
	req := httptest.NewRequest(http.MethodPatch, "/shortener/api/links/abc123/expiry", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestRouter_LinkRoutes_DeleteRequiresUserIDHeader(t *testing.T) {
	r := newRouter(t)
	req := httptest.NewRequest(http.MethodDelete, "/shortener/api/links/abc123", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// --- 404 for unknown routes ---

func TestRouter_UnknownRoute_Returns404(t *testing.T) {
	r := newRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/does-not-exist", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// --- Prometheus middleware: verify metrics path is excluded from shortener group ---

func TestRouter_MetricsEndpoint_NotUnderShortenerGroup(t *testing.T) {
	r := newRouter(t)
	// /metrics must be reachable without the Prometheus middleware adding labels
	// (it sits on the root router, not the shortenerGroup). A 200 here confirms
	// it was registered on the right group.
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	// Prometheus text exposition format
	assert.Contains(t, w.Header().Get("Content-Type"), "text/plain")
}