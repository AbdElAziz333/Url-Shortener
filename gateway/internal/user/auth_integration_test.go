package user

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"aziz.dev/gateway/internal/config"
	"aziz.dev/gateway/internal/security"
	"aziz.dev/gateway/internal/testutil"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// buildStack wires up the full user auth stack against real infra.
func buildStack(t *testing.T, db *gorm.DB) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)

	ctx := context.Background()
	redisClient := testutil.NewRedisContainer(t, ctx)

	jwtCfg := &config.JWTConfig{
		AccessSecret:  []byte("integration-access-secret-32bytes"),
		RefreshSecret: []byte("integration-refresh-secret-32byt"),
		AccessExpiry:  900,
		RefreshExpiry: 86400,
	}
	jwtSvc, err := security.NewService(jwtCfg, redisClient)
	require.NoError(t, err)

	repo := NewRepository(db)
	svc := NewService(repo, redisClient, jwtSvc)
	handler := NewHandler(svc)

	r := gin.New()
	handler.RegisterRoutes(r.Group("/"))
	return r
}

func post(t *testing.T, r *gin.Engine, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// --- Register Integration Tests ---

func TestIntegration_Register_Success(t *testing.T) {
	db := setupIntegrationDB(t)
	r := buildStack(t, db)

	w := post(t, r, "/auth/register", map[string]string{
		"email":    "newuser@example.com",
		"password": "StrongPassword1!",
	})

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "User registered successfully")
}

func TestIntegration_Register_DuplicateEmail(t *testing.T) {
	db := setupIntegrationDB(t)
	r := buildStack(t, db)

	body := map[string]string{"email": "dup@example.com", "password": "pass1"}
	post(t, r, "/auth/register", body)
	w := post(t, r, "/auth/register", body)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "email already in use")
}

func TestIntegration_Register_EmptyBody(t *testing.T) {
	db := setupIntegrationDB(t)
	r := buildStack(t, db)

	req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewBufferString(""))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// --- Login Integration Tests ---

func TestIntegration_Login_Success(t *testing.T) {
	db := setupIntegrationDB(t)
	r := buildStack(t, db)

	creds := map[string]string{"email": "login@example.com", "password": "mypassword"}
	w := post(t, r, "/auth/register", creds)
	require.Equal(t, http.StatusOK, w.Code)

	w = post(t, r, "/auth/login", creds)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Logged in successfully")
	assert.NotEmpty(t, w.Header().Get("access_token"))

	// Check refresh_token cookie
	cookies := w.Result().Cookies()
	var refreshCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "refresh_token" {
			refreshCookie = c
		}
	}
	require.NotNil(t, refreshCookie)
	assert.NotEmpty(t, refreshCookie.Value)
	assert.True(t, refreshCookie.HttpOnly)
}

func TestIntegration_Login_WrongPassword(t *testing.T) {
	db := setupIntegrationDB(t)
	r := buildStack(t, db)

	post(t, r, "/auth/register", map[string]string{"email": "pwtest@example.com", "password": "correct"})
	w := post(t, r, "/auth/login", map[string]string{"email": "pwtest@example.com", "password": "wrong"})

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid password")
}

func TestIntegration_Login_UnknownEmail(t *testing.T) {
	db := setupIntegrationDB(t)
	r := buildStack(t, db)

	w := post(t, r, "/auth/login", map[string]string{"email": "ghost@example.com", "password": "pass"})

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "user not found")
}

// --- Logout Integration Tests ---

func TestIntegration_Logout_ClearsTokenHeaders(t *testing.T) {
	db := setupIntegrationDB(t)
	r := buildStack(t, db)

	req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Empty(t, w.Header().Get("access_token"))
}

// --- Full Auth Flow ---

func TestIntegration_FullAuthFlow(t *testing.T) {
	db := setupIntegrationDB(t)
	r := buildStack(t, db)

	creds := map[string]string{"email": "fullflow@example.com", "password": "FlowPass123!"}

	// 1. Register
	w := post(t, r, "/auth/register", creds)
	require.Equal(t, http.StatusOK, w.Code, "register should succeed")

	// 2. Login
	w = post(t, r, "/auth/login", creds)
	require.Equal(t, http.StatusOK, w.Code, "login should succeed")
	accessToken := w.Header().Get("access_token")
	assert.NotEmpty(t, accessToken)

	// 3. Logout
	w = post(t, r, "/auth/logout", nil)
	assert.Equal(t, http.StatusOK, w.Code, "logout should succeed")
	assert.Contains(t, w.Body.String(), "Logged out Successfully")
}

// --- Helpers ---

func setupIntegrationDB(t *testing.T) *gorm.DB {
	t.Helper()
	ctx := context.Background()
	db := testutil.NewPostgres(t, ctx)
	err := db.AutoMigrate(&User{})
	require.NoError(t, err)
	return db
}