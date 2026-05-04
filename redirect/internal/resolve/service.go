package resolve

import (
	"context"

	"github.com/redis/go-redis/v9"

	"aziz.dev/redirect/internal/kafka"
)

type Service interface {
	ResolveCode(ctx context.Context) ([]Dto, error)
}

type service struct {
	repo       Repository
	redisClient *redis.Client
	kafkaProducer *kafka.Producer
}

func NewService(repo Repository, redisClient *redis.Client, kafkaProducer *kafka.Producer) Service {
	return &service{
		repo:       repo,
		redisClient: redisClient,
		kafkaProducer: kafkaProducer,
	}
}

func (s *service) ResolveCode(ctx context.Context) ([]Dto, error) {
	return nil, nil
}