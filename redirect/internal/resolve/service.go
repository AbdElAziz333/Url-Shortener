package resolve

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

var ErrLinkInactive = errors.New("link is not active or expired")

type RedisClient interface {
	Get(ctx context.Context, key string) *redis.StringCmd
	Set(ctx context.Context, key string, value any, expiration time.Duration) *redis.StatusCmd
}

type KafkaProducer interface {
	SendEvent(ctx context.Context, topic string, message map[string]any) error
}

type Service interface {
	ResolveCode(ctx context.Context, code string) (*Dto, error)
}

type service struct {
	repo       Repository
	cache RedisClient
	kafkaProducer KafkaProducer
}

func NewService(repo Repository, cache RedisClient, kafkaProducer KafkaProducer) Service {
	return &service{
		repo:          repo,
		cache:         cache,
		kafkaProducer: kafkaProducer,
	}
}

func (s *service) ResolveCode(ctx context.Context, code string) (*Dto, error) {
	// L1: Redis cache
	if url, err := s.cache.Get(ctx, code).Result(); err == nil && url != "" {
		return &Dto{Code: code, OriginalURL: url}, nil
	}

	// L2: database
	link, err := s.repo.Find(ctx, code)
	if err != nil {
		return nil, err
	}

	// Validate: must be active and not past its expiry.
	if !link.IsActive || (link.ExpiresAt != nil && link.ExpiresAt.Before(time.Now())) {
		return nil, ErrLinkInactive
	}
		
	// Fire-and-forget analytics — must not slow down the redirect.
	// A detached background context is intentional: the HTTP request context
	// may be cancelled before the goroutine runs.
	go func() {
		bgCtx := context.Background()
		if err := s.kafkaProducer.SendEvent(bgCtx, "url-clicked", map[string]any{
			"code":         link.Code,
			"original_url": link.OriginalURL,
			"user_id":      link.UserID,
			"clicked_at":   time.Now().UTC(),
		}); err != nil {
			log.Printf("warn: failed to send click event for code %s: %v", code, err)
		}
	}()

	// Write-back to cache with a TTL that matches the link's remaining lifetime.
	if link.ExpiresAt != nil {
		if ttl := time.Until(*link.ExpiresAt); ttl > 0 {
			// Best-effort — a cache write failure must never break the redirect.
			if err := s.cache.Set(ctx, code, link.OriginalURL, ttl).Err(); err != nil {
				log.Printf("warn: failed to cache code %s: %v", code, err)
			}
		}
	}

	return &Dto{
		Code:        link.Code,
		OriginalURL: link.OriginalURL,
	}, nil
}