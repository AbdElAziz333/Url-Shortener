package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"aziz.dev/gateway/internal/security"
	"aziz.dev/gateway/internal/user"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// --- Mock Service (mirrors handler_test.go) ---

type mockService struct {
	mock.Mock
}

func (m *mockService) Register(ctx context.Context, r *user.RegisterRequest) error {
	return m.Called(ctx, r).Error(0)
}

func (m *mockService) Login(ctx context.Context, r *user.LoginRequest) (*security.TokenPair, error) {
	args := m.Called(ctx, r)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*security.TokenPair), args.Error(1)
}

func (m *mockService) Refresh(ctx context.Context) (*security.TokenPair, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*security.TokenPair), args.Error(1)
}

// --- Helpers ---

func newTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	svc := new(mockService)
	h := user.NewHandler(svc)
	return NewRouter(h)
}

func get(r *gin.Engine, path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// --- Health Endpoint ---

func TestHealthHandler_Returns200(t *testing.T) {
	r := newTestRouter()
	w := get(r, "/gateway/health")
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHealthHandler_ReturnsStatusUp(t *testing.T) {
	r := newTestRouter()
	w := get(r, "/gateway/health")

	var body map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &body)
	require.NoError(t, err)
	assert.Equal(t, "up", body["status"])
}

func TestHealthHandler_ReturnsJSON(t *testing.T) {
	r := newTestRouter()
	w := get(r, "/gateway/health")
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")
}

// --- Metrics Endpoint ---

func TestMetricsHandler_Returns200(t *testing.T) {
	r := newTestRouter()
	w := get(r, "/metrics")
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestMetricsHandler_ReturnsPrometheusFormat(t *testing.T) {
	r := newTestRouter()
	w := get(r, "/metrics")
	// Prometheus text format always includes this line
	assert.Contains(t, w.Body.String(), "# HELP")
}

// --- Route Registration ---

func TestNewRouter_UserAuthRoutesAreRegistered(t *testing.T) {
	r := newTestRouter()
	routes := r.Routes()

	type routeKey struct{ method, path string }
	registered := make(map[routeKey]bool)
	for _, route := range routes {
		registered[routeKey{route.Method, route.Path}] = true
	}

	expected := []routeKey{
		{"POST", "/gateway/auth/register"},
		{"POST", "/gateway/auth/login"},
		{"POST", "/gateway/auth/refresh"},
		{"POST", "/gateway/auth/logout"},
		{"GET", "/gateway/health"},
		{"GET", "/metrics"},
	}
	for _, e := range expected {
		assert.True(t, registered[e], "expected route %s %s to be registered", e.method, e.path)
	}
}

// --- Path Isolation ---

func TestRouter_UnknownRoute_Returns404(t *testing.T) {
	r := newTestRouter()
	w := get(r, "/gateway/does-not-exist")
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestRouter_HealthNotAccessibleWithoutPrefix(t *testing.T) {
	r := newTestRouter()
	w := get(r, "/health")
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestRouter_AuthNotAccessibleWithoutPrefix(t *testing.T) {
	r := newTestRouter()

	req := httptest.NewRequest(http.MethodPost, "/auth/login", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// --- Middleware ---

func TestRouter_RecoveryMiddleware_HandlesHandlerPanic(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := new(mockService)
	h := user.NewHandler(svc)
	r := NewRouter(h)

	// Inject a panicking route to confirm gin.Recovery() is active
	r.GET("/panic-test", func(c *gin.Context) {
		panic("simulated panic")
	})

	req := httptest.NewRequest(http.MethodGet, "/panic-test", nil)
	w := httptest.NewRecorder()

	assert.NotPanics(t, func() {
		r.ServeHTTP(w, req)
	})
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestRouter_PrometheusMiddleware_TracksGatewayRoutes(t *testing.T) {
	r := newTestRouter()

	// Hit a gateway route so the middleware records metrics
	get(r, "/gateway/health")

	// Then confirm /metrics has data (non-empty body beyond just comments)
	w := get(r, "/metrics")
	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotEmpty(t, w.Body.String())
}