package user

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"aziz.dev/gateway/internal/security"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// --- Mock Service ---

type MockService struct {
	mock.Mock
}

func (m *MockService) Register(ctx context.Context, r *RegisterRequest) error {
	args := m.Called(ctx, r)
	return args.Error(0)
}

func (m *MockService) Login(ctx context.Context, r *LoginRequest) (*security.TokenPair, error) {
	args := m.Called(ctx, r)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*security.TokenPair), args.Error(1)
}

func (m *MockService) Refresh(ctx context.Context) (*security.TokenPair, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*security.TokenPair), args.Error(1)
}

// --- Test Helpers ---

func setupRouter(svc Service) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewHandler(svc)
	group := r.Group("/")
	h.RegisterRoutes(group)
	return r
}

func toJSON(t *testing.T, v any) *bytes.Buffer {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("failed to marshal JSON: %v", err)
	}
	return bytes.NewBuffer(b)
}

// --- Register Tests ---

func TestRegister_Success(t *testing.T) {
	svc := new(MockService)
	svc.On("Register", mock.Anything, &RegisterRequest{
		Email:    "user@example.com",
		Password: "password123",
	}).Return(nil)

	r := setupRouter(svc)
	body := toJSON(t, map[string]string{"email": "user@example.com", "password": "password123"})
	req := httptest.NewRequest(http.MethodPost, "/auth/register", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "User registered successfully")
	svc.AssertExpectations(t)
}

func TestRegister_InvalidJSON(t *testing.T) {
	svc := new(MockService)
	r := setupRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewBufferString(`not-json`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	svc.AssertNotCalled(t, "Register")
}

func TestRegister_ServiceError(t *testing.T) {
	svc := new(MockService)
	svc.On("Register", mock.Anything, mock.Anything).Return(errors.New("email already in use"))

	r := setupRouter(svc)
	body := toJSON(t, map[string]string{"email": "dup@example.com", "password": "pass"})
	req := httptest.NewRequest(http.MethodPost, "/auth/register", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "email already in use")
	svc.AssertExpectations(t)
}

// --- Login Tests ---

func TestLogin_Success(t *testing.T) {
	svc := new(MockService)
	tokenPair := &security.TokenPair{
		AccessToken:  "access-abc",
		RefreshToken: "refresh-xyz",
	}
	svc.On("Login", mock.Anything, &LoginRequest{
		Email:    "user@example.com",
		Password: "password123",
	}).Return(tokenPair, nil)

	r := setupRouter(svc)
	body := toJSON(t, map[string]string{"email": "user@example.com", "password": "password123"})
	req := httptest.NewRequest(http.MethodPost, "/auth/login", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "access-abc", w.Header().Get("access_token"))
	assert.Contains(t, w.Body.String(), "Logged in successfully")

	// Verify refresh_token cookie is set
	cookies := w.Result().Cookies()
	var refreshCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "refresh_token" {
			refreshCookie = c
			break
		}
	}
	assert.NotNil(t, refreshCookie)
	assert.Equal(t, "refresh-xyz", refreshCookie.Value)
	assert.True(t, refreshCookie.HttpOnly)

	svc.AssertExpectations(t)
}

func TestLogin_InvalidJSON(t *testing.T) {
	svc := new(MockService)
	r := setupRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewBufferString(`{bad`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	svc.AssertNotCalled(t, "Login")
}

func TestLogin_ServiceError(t *testing.T) {
	svc := new(MockService)
	svc.On("Login", mock.Anything, mock.Anything).Return(nil, errors.New("invalid password"))

	r := setupRouter(svc)
	body := toJSON(t, map[string]string{"email": "user@example.com", "password": "wrong"})
	req := httptest.NewRequest(http.MethodPost, "/auth/login", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid password")
	svc.AssertExpectations(t)
}

// --- Refresh Tests ---

func TestRefresh_Success(t *testing.T) {
	svc := new(MockService)
	svc.On("Refresh", mock.Anything).Return(&security.TokenPair{
		AccessToken:  "new-access",
		RefreshToken: "new-refresh",
	}, nil)

	r := setupRouter(svc)
	req := httptest.NewRequest(http.MethodPost, "/auth/refresh", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "new-access", w.Header().Get("access_token"))
	assert.Contains(t, w.Body.String(), "Refreshed token successfully")
	svc.AssertExpectations(t)
}

func TestRefresh_ServiceError(t *testing.T) {
	svc := new(MockService)
	svc.On("Refresh", mock.Anything).Return(nil, errors.New("token expired"))

	r := setupRouter(svc)
	req := httptest.NewRequest(http.MethodPost, "/auth/refresh", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "token expired")
	svc.AssertExpectations(t)
}

// --- Logout Tests ---

func TestLogout_Success(t *testing.T) {
	svc := new(MockService)
	r := setupRouter(svc)

	req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Logged out Successfully")
	assert.Equal(t, "", w.Header().Get("access_token"))
	assert.Equal(t, "", w.Header().Get("refresh_token"))
}

// --- Route Registration Tests ---

func TestRegisterRoutes_AllRoutesExist(t *testing.T) {
	svc := new(MockService)
	r := setupRouter(svc)

	routes := r.Routes()
	paths := make(map[string]bool)
	for _, route := range routes {
		paths[route.Method+":"+route.Path] = true
	}

	assert.True(t, paths["POST:/auth/register"], "POST /auth/register should be registered")
	assert.True(t, paths["POST:/auth/login"], "POST /auth/login should be registered")
	assert.True(t, paths["POST:/auth/refresh"], "POST /auth/refresh should be registered")
	assert.True(t, paths["POST:/auth/logout"], "POST /auth/logout should be registered")
}