package auth

import (
	"fmt"

	"aziz.dev/gateway/internal/config"
	"aziz.dev/gateway/internal/security"
	"github.com/golang-jwt/jwt/v5"
)

func ValidateAccessToken(cfg *config.JWTConfig, token string) bool {
	return validateToken(token, cfg.AccessSecret) == nil
}

func ValidateRefreshToken(cfg *config.JWTConfig, token string) bool {
	return validateToken(token, cfg.RefreshSecret) == nil
}

func validateToken(token string, secret []byte) error {
	_, err := jwt.ParseWithClaims(token, &security.Claim{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return secret, nil
	})
	return err
}
