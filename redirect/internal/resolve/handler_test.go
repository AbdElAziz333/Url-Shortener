package resolve

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Mock Service
// ---------------------------------------------------------------------------

type mockService struct {
	mock.Mock
}

func (m *mockService) ResolveCode(ctx context.Context, code string) (*Dto, error) {
	args := m.Called(ctx, code)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Dto), args.Error(1)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func setupRouter(svc Service) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewHandler(svc)
	group := r.Group("/r")
	h.RegisterRoutes(group)
	return r
}

func performRequest(r *gin.Engine, method, path string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, nil)
	r.ServeHTTP(w, req)
	return w
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestHandler_ResolveCode_Success(t *testing.T) {
	svc := new(mockService)
	svc.On("ResolveCode", mock.Anything, "abc123").
		Return(&Dto{Code: "abc123", OriginalURL: "https://example.com"}, nil)

	r := setupRouter(svc)
	w := performRequest(r, http.MethodGet, "/r/abc123")

	assert.Equal(t, http.StatusFound, w.Code)
	assert.Equal(t, "https://example.com", w.Header().Get("Location"))
	svc.AssertExpectations(t)
}

func TestHandler_ResolveCode_NotFound(t *testing.T) {
	svc := new(mockService)
	svc.On("ResolveCode", mock.Anything, "missing").
		Return(nil, ErrNotFound)

	r := setupRouter(svc)
	w := performRequest(r, http.MethodGet, "/r/missing")

	assert.Equal(t, http.StatusNotFound, w.Code)
	svc.AssertExpectations(t)
}

func TestHandler_ResolveCode_LinkInactive(t *testing.T) {
	svc := new(mockService)
	svc.On("ResolveCode", mock.Anything, "expired").
		Return(nil, ErrLinkInactive)

	r := setupRouter(svc)
	w := performRequest(r, http.MethodGet, "/r/expired")

	assert.Equal(t, http.StatusGone, w.Code)
	svc.AssertExpectations(t)
}

func TestHandler_ResolveCode_GenericServiceError(t *testing.T) {
	svc := new(mockService)
	svc.On("ResolveCode", mock.Anything, "bad").
		Return(nil, errors.New("unexpected db error"))

	r := setupRouter(svc)
	w := performRequest(r, http.MethodGet, "/r/bad")

	// Any non-ErrLinkInactive error maps to 404 per handler logic.
	assert.Equal(t, http.StatusNotFound, w.Code)
	svc.AssertExpectations(t)
}

func TestHandler_RegisterRoutes_RoutesAreRegistered(t *testing.T) {
	svc := new(mockService)
	svc.On("ResolveCode", mock.Anything, "x").
		Return(&Dto{Code: "x", OriginalURL: "https://x.com"}, nil)

	r := setupRouter(svc)

	// Verify a registered route actually responds (not 404 from the router).
	w := performRequest(r, http.MethodGet, "/r/x")
	require.NotEqual(t, http.StatusNotFound, w.Code,
		"route /r/:code should be registered")
}

func TestHandler_ResolveCode_ResponseBodyOnError(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		expectedStatus int
		bodyContains   string
	}{
		{
			name:           "not found error body",
			err:            ErrNotFound,
			expectedStatus: http.StatusNotFound,
			bodyContains:   "error",
		},
		{
			name:           "inactive link error body",
			err:            ErrLinkInactive,
			expectedStatus: http.StatusGone,
			bodyContains:   "error",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := new(mockService)
			svc.On("ResolveCode", mock.Anything, "code").Return(nil, tc.err)

			r := setupRouter(svc)
			w := performRequest(r, http.MethodGet, "/r/code")

			assert.Equal(t, tc.expectedStatus, w.Code)
			assert.Contains(t, w.Body.String(), tc.bodyContains)
			svc.AssertExpectations(t)
		})
	}
}