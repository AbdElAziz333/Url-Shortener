package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"aziz.dev/redirect/internal/resolve"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// ---------------------------------------------------------------------------
// Mock Service — minimal stub so resolve.Handler can be constructed
// ---------------------------------------------------------------------------

type mockResolveService struct {
	mock.Mock
}

func (m *mockResolveService) ResolveCode(ctx context.Context, code string) (*resolve.Dto, error) {
	args := m.Called(ctx, code)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*resolve.Dto), args.Error(1)
}

// ---------------------------------------------------------------------------
// Helper
// ---------------------------------------------------------------------------

func newTestRouter(svc resolve.Service) *gin.Engine {
	h := resolve.NewHandler(svc)
	return NewRouter(h)
}

func do(r *gin.Engine, method, path string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, nil)
	r.ServeHTTP(w, req)
	return w
}

// ---------------------------------------------------------------------------
// Health endpoint
// ---------------------------------------------------------------------------

func TestRouter_Health_ReturnsOK(t *testing.T) {
	r := newTestRouter(new(mockResolveService))
	w := do(r, http.MethodGet, "/redirect/health")

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRouter_Health_ReturnsUpStatus(t *testing.T) {
	r := newTestRouter(new(mockResolveService))
	w := do(r, http.MethodGet, "/redirect/health")

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "up", body["status"])
}

func TestRouter_Health_ContentTypeIsJSON(t *testing.T) {
	r := newTestRouter(new(mockResolveService))
	w := do(r, http.MethodGet, "/redirect/health")

	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")
}

// ---------------------------------------------------------------------------
// Metrics endpoint
// ---------------------------------------------------------------------------

func TestRouter_Metrics_ReturnsOK(t *testing.T) {
	r := newTestRouter(new(mockResolveService))
	w := do(r, http.MethodGet, "/metrics")

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRouter_Metrics_ReturnsPrometheusText(t *testing.T) {
	r := newTestRouter(new(mockResolveService))
	w := do(r, http.MethodGet, "/metrics")

	// Prometheus text format always starts with comment lines or metric lines.
	assert.Contains(t, w.Header().Get("Content-Type"), "text/plain")
}

// ---------------------------------------------------------------------------
// Resolve routes (delegated to resolve.Handler)
// ---------------------------------------------------------------------------

func TestRouter_Resolve_HitsResolveHandler(t *testing.T) {
	svc := new(mockResolveService)
	svc.On("ResolveCode", mock.Anything, "abc123").
		Return(&resolve.Dto{Code: "abc123", OriginalURL: "https://example.com"}, nil)

	r := newTestRouter(svc)
	w := do(r, http.MethodGet, "/redirect/abc123")

	assert.Equal(t, http.StatusFound, w.Code)
	assert.Equal(t, "https://example.com", w.Header().Get("Location"))
	svc.AssertExpectations(t)
}

func TestRouter_Resolve_NotFound(t *testing.T) {
	svc := new(mockResolveService)
	svc.On("ResolveCode", mock.Anything, "missing").
		Return(nil, resolve.ErrNotFound)

	r := newTestRouter(svc)
	w := do(r, http.MethodGet, "/redirect/missing")

	assert.Equal(t, http.StatusNotFound, w.Code)
	svc.AssertExpectations(t)
}

func TestRouter_Resolve_InactiveLink(t *testing.T) {
	svc := new(mockResolveService)
	svc.On("ResolveCode", mock.Anything, "expired").
		Return(nil, resolve.ErrLinkInactive)

	r := newTestRouter(svc)
	w := do(r, http.MethodGet, "/redirect/expired")

	assert.Equal(t, http.StatusGone, w.Code)
	svc.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// Unregistered routes
// ---------------------------------------------------------------------------

func TestRouter_UnknownRoute_Returns404(t *testing.T) {
	r := newTestRouter(new(mockResolveService))
	w := do(r, http.MethodGet, "/does-not-exist")

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestRouter_WrongMethod_Returns405(t *testing.T) {
	r := newTestRouter(new(mockResolveService))
	// /redirect/health is GET only.
	w := do(r, http.MethodPost, "/redirect/health")

	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

// ---------------------------------------------------------------------------
// Panic recovery middleware
// ---------------------------------------------------------------------------

func TestRouter_PanicRecovery_Returns500(t *testing.T) {
	svc := new(mockResolveService)
	svc.On("ResolveCode", mock.Anything, mock.Anything).
		Panic("simulated panic")

	r := newTestRouter(svc)
	w := do(r, http.MethodGet, "/redirect/panic-code")

	// gin.Recovery() must catch the panic and return 500 rather than crashing.
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}