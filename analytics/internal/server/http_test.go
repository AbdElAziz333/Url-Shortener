package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"aziz.dev/analytics/internal/stat"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func performRequest(r *gin.Engine, method, path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func decodeBody(t *testing.T, w *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var body map[string]any
	assert.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	return body
}

// ---------------------------------------------------------------------------
// Mock stat.Service — the minimal surface needed to construct a stat.Handler.
// ---------------------------------------------------------------------------

type mockStatService struct {
	mock.Mock
}

func (m *mockStatService) GetTotalClicks(ctx context.Context, code string) ([]stat.Dto, error) {
	args := m.Called(ctx, code)
	return args.Get(0).([]stat.Dto), args.Error(1)
}

func (m *mockStatService) GetGeo(ctx context.Context, code string) ([]stat.Dto, error) {
	args := m.Called(ctx, code)
	return args.Get(0).([]stat.Dto), args.Error(1)
}

func (m *mockStatService) GetReferrers(ctx context.Context, code string) ([]stat.Dto, error) {
	args := m.Called(ctx, code)
	return args.Get(0).([]stat.Dto), args.Error(1)
}

// newTestRouter builds the real router with a mock stat.Handler wired in.
func newTestRouter(t *testing.T) (*gin.Engine, *mockStatService) {
	t.Helper()
	svc := new(mockStatService)
	handler := stat.NewHandler(svc)
	return NewRouter(handler), svc
}

// ---------------------------------------------------------------------------
// /analytics/health
// ---------------------------------------------------------------------------

func TestHealthEndpoint_Returns200(t *testing.T) {
	r, _ := newTestRouter(t)

	w := performRequest(r, http.MethodGet, "/analytics/health")

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHealthEndpoint_ReturnsOKBody(t *testing.T) {
	r, _ := newTestRouter(t)

	w := performRequest(r, http.MethodGet, "/analytics/health")
	body := decodeBody(t, w)

	assert.Equal(t, "OK", body["message"])
}

func TestHealthEndpoint_ContentTypeIsJSON(t *testing.T) {
	r, _ := newTestRouter(t)

	w := performRequest(r, http.MethodGet, "/analytics/health")

	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")
}

func TestHealthEndpoint_MethodNotAllowed(t *testing.T) {
	r, _ := newTestRouter(t)

	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodDelete} {
		w := performRequest(r, method, "/analytics/health")
		assert.Equal(t, http.StatusNotFound, w.Code, "expected 404 for method %s", method)
	}
}

// ---------------------------------------------------------------------------
// /metrics
// ---------------------------------------------------------------------------

func TestMetricsEndpoint_Returns200(t *testing.T) {
	r, _ := newTestRouter(t)

	w := performRequest(r, http.MethodGet, "/metrics")

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestMetricsEndpoint_ReturnsPrometheusBody(t *testing.T) {
	r, _ := newTestRouter(t)

	w := performRequest(r, http.MethodGet, "/metrics")

	// Prometheus text format always starts with a comment or a metric family.
	// Just assert the body is non-empty and plaintext.
	assert.NotEmpty(t, w.Body.String())
	ct := w.Header().Get("Content-Type")
	assert.Contains(t, ct, "text/plain")
}

func TestMetricsEndpoint_MethodNotAllowed(t *testing.T) {
	r, _ := newTestRouter(t)

	w := performRequest(r, http.MethodPost, "/metrics")
	// Gin returns 404 for non-GET methods when only GET is registered.
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// ---------------------------------------------------------------------------
// Stat routes — routing / mounting correctness (handler logic is in handler_test.go)
// ---------------------------------------------------------------------------

func TestStatRoutes_TotalClicks_Routed(t *testing.T) {
	r, svc := newTestRouter(t)
	svc.On("GetTotalClicks", mock.Anything, "abc").
		Return([]stat.Dto{{Key: "total", Count: 5}}, nil)

	w := performRequest(r, http.MethodGet, "/analytics/api/stats/abc")

	assert.Equal(t, http.StatusOK, w.Code)
	svc.AssertExpectations(t)
}

func TestStatRoutes_Geo_Routed(t *testing.T) {
	r, svc := newTestRouter(t)
	svc.On("GetGeo", mock.Anything, "abc").
		Return([]stat.Dto{{Key: "US", Count: 10}}, nil)

	w := performRequest(r, http.MethodGet, "/analytics/api/stats/abc/geo")

	assert.Equal(t, http.StatusOK, w.Code)
	svc.AssertExpectations(t)
}

func TestStatRoutes_Referrers_Routed(t *testing.T) {
	r, svc := newTestRouter(t)
	svc.On("GetReferrers", mock.Anything, "abc").
		Return([]stat.Dto{{Key: "google.com", Count: 3}}, nil)

	w := performRequest(r, http.MethodGet, "/analytics/api/stats/abc/referrers")

	assert.Equal(t, http.StatusOK, w.Code)
	svc.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// 404 and unknown paths
// ---------------------------------------------------------------------------

func TestUnknownRoute_Returns404(t *testing.T) {
	r, _ := newTestRouter(t)

	for _, path := range []string{
		"/",
		"/analytics",
		"/analytics/api",
		"/analytics/api/stats",
		"/analytics/unknown",
		"/notreal",
	} {
		w := performRequest(r, http.MethodGet, path)
		assert.Equal(t, http.StatusNotFound, w.Code, "path: %s", path)
	}
}

// ---------------------------------------------------------------------------
// Middleware: Recovery
// ---------------------------------------------------------------------------

// panicHandler is a stand-alone gin handler that always panics; it lets us
// verify that gin.Recovery() catches the panic and returns a 500 rather than
// crashing the process.
func TestRecoveryMiddleware_HandlesHandlerPanic(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.GET("/panic", func(c *gin.Context) {
		panic("deliberate test panic")
	})

	w := performRequest(r, http.MethodGet, "/panic")

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}