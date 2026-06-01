package security

import (
	"testing"
	"time"

	"aziz.dev/gateway/internal/config"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validConfig() *config.JWTConfig {
	return &config.JWTConfig{
		AccessSecret:  []byte("super-secret-access-key-32-bytes!"),
		RefreshSecret: []byte("super-secret-refresh-key-32-bytes!"),
		AccessExpiry:  900,   // 15 min
		RefreshExpiry: 86400, // 24 hr
	}
}

// --- Constructor Tests ---

func TestNewService_Success(t *testing.T) {
	svc, err := NewService(validConfig(), nil)
	require.NoError(t, err)
	assert.NotNil(t, svc)
}

func TestNewService_MissingAccessSecret(t *testing.T) {
	cfg := validConfig()
	cfg.AccessSecret = []byte{}
	_, err := NewService(cfg, nil)
	assert.EqualError(t, err, "access secret key is required")
}

func TestNewService_MissingRefreshSecret(t *testing.T) {
	cfg := validConfig()
	cfg.RefreshSecret = []byte{}
	_, err := NewService(cfg, nil)
	assert.EqualError(t, err, "refresh secret key is required")
}

func TestNewService_RefreshExpirySmallerThanAccess(t *testing.T) {
	cfg := validConfig()
	cfg.RefreshExpiry = cfg.AccessExpiry - 1
	_, err := NewService(cfg, nil)
	assert.EqualError(t, err, "refresh expiry must be greater than access expiry")
}

func TestNewService_RefreshExpiryEqualToAccess(t *testing.T) {
	cfg := validConfig()
	cfg.RefreshExpiry = cfg.AccessExpiry
	_, err := NewService(cfg, nil)
	assert.EqualError(t, err, "refresh expiry must be greater than access expiry")
}

// --- GenerateAccessToken Tests ---

func TestGenerateAccessToken_ValidToken(t *testing.T) {
	cfg := validConfig()
	svc, err := NewService(cfg, nil)
	require.NoError(t, err)

	userID := uuid.New().String()
	tokenStr, err := svc.GenerateAccessToken(userID)
	require.NoError(t, err)
	assert.NotEmpty(t, tokenStr)

	// Parse and verify claims
	claims := &Claim{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		return cfg.AccessSecret, nil
	})
	require.NoError(t, err)
	assert.True(t, token.Valid)
	assert.Equal(t, userID, claims.Subject)
	assert.Equal(t, "gateway", claims.Issuer)
	assert.Equal(t, userID, claims.UserID.String())
	assert.WithinDuration(t, time.Now().Add(15*time.Minute), claims.ExpiresAt.Time, 5*time.Second)
}

func TestGenerateAccessToken_SignedWithAccessSecret(t *testing.T) {
	cfg := validConfig()
	svc, _ := NewService(cfg, nil)

	tokenStr, _ := svc.GenerateAccessToken(uuid.New().String())

	// Should fail with wrong secret
	claims := &Claim{}
	_, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		return cfg.RefreshSecret, nil
	})
	assert.Error(t, err, "access token should not validate with refresh secret")
}

// --- GenerateRefreshToken Tests ---

func TestGenerateRefreshToken_ValidToken(t *testing.T) {
	cfg := validConfig()
	svc, err := NewService(cfg, nil)
	require.NoError(t, err)

	userID := uuid.New().String()
	tokenStr, err := svc.GenerateRefreshToken(userID)
	require.NoError(t, err)
	assert.NotEmpty(t, tokenStr)

	claims := &Claim{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		return cfg.RefreshSecret, nil
	})
	require.NoError(t, err)
	assert.True(t, token.Valid)
	assert.Equal(t, userID, claims.Subject)
	assert.Equal(t, "gateway", claims.Issuer)
	assert.WithinDuration(t, time.Now().Add(7*24*time.Hour), claims.ExpiresAt.Time, 5*time.Second)
}

func TestGenerateRefreshToken_SignedWithRefreshSecret(t *testing.T) {
	cfg := validConfig()
	svc, _ := NewService(cfg, nil)

	tokenStr, _ := svc.GenerateRefreshToken(uuid.New().String())

	// Should fail with wrong secret
	claims := &Claim{}
	_, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		return cfg.AccessSecret, nil
	})
	assert.Error(t, err, "refresh token should not validate with access secret")
}

func TestGenerateTokens_AccessAndRefreshAreDifferent(t *testing.T) {
	cfg := validConfig()
	svc, _ := NewService(cfg, nil)
	userID := uuid.New().String()

	access, _ := svc.GenerateAccessToken(userID)
	refresh, _ := svc.GenerateRefreshToken(userID)

	assert.NotEqual(t, access, refresh)
}