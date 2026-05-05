package security

import (
	"errors"
	"time"

	"aziz.dev/gateway/internal/config"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type Service interface {
	GenerateAccessToken(userId string) (string, error)
	GenerateRefreshToken(userId string) (string, error)
}

type Claim struct {
	UserID uuid.UUID `json:"user_id"`
	jwt.RegisteredClaims
}

type TokenPair struct {
	AccessToken string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type service struct {
	config *config.JWTConfig
	redisClient *redis.Client
}

func NewService(cfg *config.JWTConfig, redisClient *redis.Client) (*service, error) {
	if len(cfg.AccessSecret) == 0 {
		return nil, errors.New("access secret key is required")
	}

	if len(cfg.RefreshSecret) == 0 {
		return nil, errors.New("refresh secret key is required")
	}

	if cfg.RefreshExpiry <= cfg.AccessExpiry {
		return nil, errors.New("refresh expiry must be greater than access expiry")
	}

	return &service{
		config:      cfg,
		redisClient: redisClient,
	}, nil
}

func (s *service) GenerateAccessToken(userID string) (string, error) {
	claim := Claim{
		UserID: uuid.Must(uuid.Parse(userID)),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute * 15)),
			IssuedAt: jwt.NewNumericDate(time.Now()),
			Issuer: "gateway",
			Subject: userID,
		},
	}
	
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claim)
	return token.SignedString(s.config.AccessSecret)
}

func (s *service) GenerateRefreshToken(userID string) (string, error) {
	claim := Claim{
		UserID: uuid.Must(uuid.Parse(userID)),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 24 * 7)),
			IssuedAt: jwt.NewNumericDate(time.Now()),
			Issuer: "gateway",
			Subject: userID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claim)
	return token.SignedString(s.config.RefreshSecret)
}