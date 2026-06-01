package stat

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockService struct {
	mock.Mock
}

func (m *mockService) GetTotalClicks(ctx context.Context, code string) ([]Dto, error) {
	args := m.Called(ctx, code)
	return args.Get(0).([]Dto), args.Error(1)
}

func (m *mockService) GetGeo(ctx context.Context, code string) ([]Dto, error) {
	args := m.Called(ctx, code)
	return args.Get(0).([]Dto), args.Error(1)
}

func (m *mockService) GetReferrers(ctx context.Context, code string) ([]Dto, error) {
	args := m.Called(ctx, code)
	return args.Get(0).([]Dto), args.Error(1)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func setupRouter(svc Service) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewHandler(svc)
	g := r.Group("/analytics/api/stats")
	h.RegisterRoutes(g)
	return r
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
// GetTotalClicks
// ---------------------------------------------------------------------------

func TestGetTotalClicks_Success(t *testing.T) {
	svc := new(mockService)
	expected := []Dto{{Key: "total", Count: 42}}
	svc.On("GetTotalClicks", mock.Anything, "abc123").Return(expected, nil)

	w := performRequest(setupRouter(svc), http.MethodGet, "/analytics/api/stats/abc123")

	assert.Equal(t, http.StatusOK, w.Code)
	body := decodeBody(t, w)
	data := body["data"].([]any)
	assert.Len(t, data, 1)
	assert.Equal(t, "total", data[0].(map[string]any)["key"])
	assert.Equal(t, float64(42), data[0].(map[string]any)["count"])
	svc.AssertExpectations(t)
}

func TestGetTotalClicks_ServiceError(t *testing.T) {
	svc := new(mockService)
	svc.On("GetTotalClicks", mock.Anything, "abc123").
		Return([]Dto{}, errors.New("db error"))

	w := performRequest(setupRouter(svc), http.MethodGet, "/analytics/api/stats/abc123")

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	body := decodeBody(t, w)
	assert.Equal(t, "db error", body["error"])
	svc.AssertExpectations(t)
}

func TestGetTotalClicks_EmptyResult(t *testing.T) {
	svc := new(mockService)
	svc.On("GetTotalClicks", mock.Anything, "xyz").Return([]Dto{}, nil)

	w := performRequest(setupRouter(svc), http.MethodGet, "/analytics/api/stats/xyz")

	assert.Equal(t, http.StatusOK, w.Code)
	body := decodeBody(t, w)
	assert.Empty(t, body["data"])
	svc.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// GetGeo
// ---------------------------------------------------------------------------

func TestGetGeo_Success(t *testing.T) {
	svc := new(mockService)
	expected := []Dto{
		{Key: "US", Count: 100},
		{Key: "DE", Count: 50},
	}
	svc.On("GetGeo", mock.Anything, "abc123").Return(expected, nil)

	w := performRequest(setupRouter(svc), http.MethodGet, "/analytics/api/stats/abc123/geo")

	assert.Equal(t, http.StatusOK, w.Code)
	body := decodeBody(t, w)
	data := body["data"].([]any)
	assert.Len(t, data, 2)
	assert.Equal(t, "US", data[0].(map[string]any)["key"])
	assert.Equal(t, float64(100), data[0].(map[string]any)["count"])
	svc.AssertExpectations(t)
}

func TestGetGeo_ServiceError(t *testing.T) {
	svc := new(mockService)
	svc.On("GetGeo", mock.Anything, "abc123").
		Return([]Dto{}, errors.New("mongo unavailable"))

	w := performRequest(setupRouter(svc), http.MethodGet, "/analytics/api/stats/abc123/geo")

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	body := decodeBody(t, w)
	assert.Equal(t, "mongo unavailable", body["error"])
	svc.AssertExpectations(t)
}

func TestGetGeo_EmptyResult(t *testing.T) {
	svc := new(mockService)
	svc.On("GetGeo", mock.Anything, "nodata").Return([]Dto{}, nil)

	w := performRequest(setupRouter(svc), http.MethodGet, "/analytics/api/stats/nodata/geo")

	assert.Equal(t, http.StatusOK, w.Code)
	body := decodeBody(t, w)
	assert.Empty(t, body["data"])
	svc.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// GetReferrers
// ---------------------------------------------------------------------------

func TestGetReferrers_Success(t *testing.T) {
	svc := new(mockService)
	expected := []Dto{
		{Key: "google.com", Count: 200},
		{Key: "twitter.com", Count: 80},
	}
	svc.On("GetReferrers", mock.Anything, "abc123").Return(expected, nil)

	w := performRequest(setupRouter(svc), http.MethodGet, "/analytics/api/stats/abc123/referrers")

	assert.Equal(t, http.StatusOK, w.Code)
	body := decodeBody(t, w)
	data := body["data"].([]any)
	assert.Len(t, data, 2)
	assert.Equal(t, "google.com", data[0].(map[string]any)["key"])
	svc.AssertExpectations(t)
}

func TestGetReferrers_ServiceError(t *testing.T) {
	svc := new(mockService)
	svc.On("GetReferrers", mock.Anything, "abc123").
		Return([]Dto{}, errors.New("timeout"))

	w := performRequest(setupRouter(svc), http.MethodGet, "/analytics/api/stats/abc123/referrers")

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	body := decodeBody(t, w)
	assert.Equal(t, "timeout", body["error"])
	svc.AssertExpectations(t)
}

func TestGetReferrers_UnknownReferrer(t *testing.T) {
	svc := new(mockService)
	expected := []Dto{{Key: "unknown", Count: 5}}
	svc.On("GetReferrers", mock.Anything, "abc123").Return(expected, nil)

	w := performRequest(setupRouter(svc), http.MethodGet, "/analytics/api/stats/abc123/referrers")

	assert.Equal(t, http.StatusOK, w.Code)
	body := decodeBody(t, w)
	data := body["data"].([]any)
	assert.Equal(t, "unknown", data[0].(map[string]any)["key"])
	svc.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// Route registration smoke tests
// ---------------------------------------------------------------------------

func TestRoutes_NotFound(t *testing.T) {
	svc := new(mockService)
	r := setupRouter(svc)

	w := performRequest(r, http.MethodGet, "/analytics/api/stats/")
	// Gin returns 301 redirect or 404 for trailing slash with no param
	assert.True(t, w.Code == http.StatusNotFound || w.Code == http.StatusMovedPermanently)
}

func TestRoutes_MethodNotAllowed(t *testing.T) {
	svc := new(mockService)
	// No service call expected for wrong method
	r := setupRouter(svc)

	w := performRequest(r, http.MethodPost, "/analytics/api/stats/abc123")
	assert.Equal(t, http.StatusNotFound, w.Code)
}