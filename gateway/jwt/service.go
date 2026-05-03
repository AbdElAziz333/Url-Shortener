package jwt

import "github.com/redis/go-redis/v9"

type Service interface {
	GenerateAccessToken() string
	GenerateRefreshToken() string
	ValidateAccessToken() bool
	ValidateRefreshToken() bool
}

type service struct {
	redisClient *redis.Client
}

func NewService(redisClient *redis.Client) Service {
	return &service{
		redisClient: redisClient,
	}
}

func (s *service) GenerateAccessToken() string {
	return ""
}

func (s *service) GenerateRefreshToken() string {
	return ""
}

func (s *service) ValidateAccessToken() bool {
	return false
}

func (s *service) ValidateRefreshToken() bool {
	return false
}
