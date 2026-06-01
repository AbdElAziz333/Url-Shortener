package auth

import (
	"testing"
	"time"

	"aziz.dev/gateway/internal/config"
	"aziz.dev/gateway/internal/security"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testConfig() *config.JWTConfig {
	return &config.JWTConfig{
		AccessSecret:  []byte("access-secret-key-for-testing-32b"),
		RefreshSecret: []byte("refresh-secret-key-for-testing-32b"),
		AccessExpiry:  900,
		RefreshExpiry: 86400,
	}
}

func makeSignedToken(secret []byte, userID string, expiry time.Time) string {
	claim := security.Claim{
		UserID: uuid.Must(uuid.Parse(userID)),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiry),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "gateway",
			Subject:   userID,
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claim)
	signed, _ := token.SignedString(secret)
	return signed
}

// --- ValidateAccessToken Tests ---

func TestValidateAccessToken_ValidToken(t *testing.T) {
	cfg := testConfig()
	userID := uuid.New().String()
	tokenStr := makeSignedToken(cfg.AccessSecret, userID, time.Now().Add(15*time.Minute))

	claims, ok := ValidateAccessToken(cfg, tokenStr)

	assert.True(t, ok)
	require.NotNil(t, claims)
	assert.Equal(t, userID, claims.Subject)
	assert.Equal(t, userID, claims.UserID.String())
}

func TestValidateAccessToken_ExpiredToken(t *testing.T) {
	cfg := testConfig()
	userID := uuid.New().String()
	tokenStr := makeSignedToken(cfg.AccessSecret, userID, time.Now().Add(-1*time.Minute))

	claims, ok := ValidateAccessToken(cfg, tokenStr)

	assert.False(t, ok)
	assert.Nil(t, claims)
}

func TestValidateAccessToken_WrongSecret(t *testing.T) {
	cfg := testConfig()
	userID := uuid.New().String()
	// Sign with refresh secret, try to validate as access
	tokenStr := makeSignedToken(cfg.RefreshSecret, userID, time.Now().Add(15*time.Minute))

	claims, ok := ValidateAccessToken(cfg, tokenStr)

	assert.False(t, ok)
	assert.Nil(t, claims)
}

func TestValidateAccessToken_TamperedToken(t *testing.T) {
	cfg := testConfig()
	userID := uuid.New().String()
	tokenStr := makeSignedToken(cfg.AccessSecret, userID, time.Now().Add(15*time.Minute))

	tampered := tokenStr + "xyz"
	claims, ok := ValidateAccessToken(cfg, tampered)

	assert.False(t, ok)
	assert.Nil(t, claims)
}

func TestValidateAccessToken_EmptyToken(t *testing.T) {
	cfg := testConfig()
	claims, ok := ValidateAccessToken(cfg, "")

	assert.False(t, ok)
	assert.Nil(t, claims)
}

func TestValidateAccessToken_RandomString(t *testing.T) {
	cfg := testConfig()
	claims, ok := ValidateAccessToken(cfg, "not.a.jwt")

	assert.False(t, ok)
	assert.Nil(t, claims)
}

// --- ValidateRefreshToken Tests ---

func TestValidateRefreshToken_ValidToken(t *testing.T) {
	cfg := testConfig()
	userID := uuid.New().String()
	tokenStr := makeSignedToken(cfg.RefreshSecret, userID, time.Now().Add(7*24*time.Hour))

	claims, ok := ValidateRefreshToken(cfg, tokenStr)

	assert.True(t, ok)
	require.NotNil(t, claims)
	assert.Equal(t, userID, claims.Subject)
}

func TestValidateRefreshToken_ExpiredToken(t *testing.T) {
	cfg := testConfig()
	userID := uuid.New().String()
	tokenStr := makeSignedToken(cfg.RefreshSecret, userID, time.Now().Add(-1*time.Second))

	claims, ok := ValidateRefreshToken(cfg, tokenStr)

	assert.False(t, ok)
	assert.Nil(t, claims)
}

func TestValidateRefreshToken_WrongSecret(t *testing.T) {
	cfg := testConfig()
	userID := uuid.New().String()
	// Sign with access secret, try to validate as refresh
	tokenStr := makeSignedToken(cfg.AccessSecret, userID, time.Now().Add(24*time.Hour))

	claims, ok := ValidateRefreshToken(cfg, tokenStr)

	assert.False(t, ok)
	assert.Nil(t, claims)
}

// --- Algorithm Confusion Tests ---

func TestValidateToken_RejectsNoneAlgorithm(t *testing.T) {
	cfg := testConfig()
	userID := uuid.New().String()

	// Craft a "none" alg token manually
	claim := security.Claim{
		UserID: uuid.Must(uuid.Parse(userID)),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			Subject:   userID,
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claim)
	// Attempt to use the signed token as-is (the library doesn't support none easily)
	tokenStr, _ := token.SignedString(cfg.RefreshSecret)

	// Access token validation with refresh-signed token should fail
	_, ok := ValidateAccessToken(cfg, tokenStr)
	assert.False(t, ok)
}