package resolve

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/redis/go-redis/v9"

	"aziz.dev/redirect/internal/kafka"
)

type Service interface {
	ResolveCode(ctx context.Context, code string) (*Dto, error)
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

func (s *service) ResolveCode(ctx context.Context, code string) (*Dto, error) {
	// look up the code in Redis cache (L1)
	cache, err := s.redisClient.Get(ctx, code).Result()
	if err == nil && cache != "" {
		// TODO: Return the link
		return &Dto{
			Code: code,
			OriginalURL: cache,
		}, nil
	}

	// fallback to the database if cache miss
	link, err := s.repo.Find(ctx, code)
	if err != nil {
		return nil, err
	}

	// validate the url isn't expired or disabled
	if !link.IsActive || link.ExpiresAt.Before(time.Now()) {
		return nil, errors.New("link is not active or expired")
	}
		
	// Write a fire-and-forget event for this click to kafka without waiting for a response 
	// so it never slows down the redirect
    // Fire-and-forget analytics
	go func() {
		if err := s.kafkaProducer.SendEvent(ctx, "url-clicked", map[string]any{
			"code":        link.Code,
			"original_url": link.OriginalURL,
			"user_id":     link.UserID,
			"clicked_at":  time.Now(),
		}); err != nil {
			log.Printf("warn: failed to send click event for code %s: %v", code, err)
		}
	}()

	// store it to redis
	ttl := time.Until(*link.ExpiresAt)
	if ttl > 0 {
		s.redisClient.Set(ctx, code, link.OriginalURL, ttl)
	}

	// increment counter 

	return &Dto{
		Code:        link.Code,
		OriginalURL: link.OriginalURL,
	}, nil
}