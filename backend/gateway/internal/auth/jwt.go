package auth

import (
	"fmt"

	"aziz.dev/gateway/internal/config"
	"aziz.dev/gateway/internal/security"
	"github.com/golang-jwt/jwt/v5"
)

func ValidateAccessToken(cfg *config.JWTConfig, token string) (*security.Claim, bool) {
	claims, err := validateToken(token, cfg.AccessSecret)
	return claims, err == nil
}

func ValidateRefreshToken(cfg *config.JWTConfig, token string) (*security.Claim, bool) {
	claims, err := validateToken(token, cfg.RefreshSecret)
	return claims, err == nil
}

func validateToken(token string, secret []byte) (*security.Claim, error) {
	parsedToken, err := jwt.ParseWithClaims(token, &security.Claim{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return secret, nil
	})
	if err != nil {
		return nil, err
	}
	
	if claims, ok := parsedToken.Claims.(*security.Claim); ok && parsedToken.Valid {
		return claims, nil
	}
	return nil, fmt.Errorf("invalid token")
}
