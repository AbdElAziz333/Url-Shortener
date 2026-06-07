package resolve

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
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
	repo          Repository
	cache         RedisClient
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
	logrus.WithField("code", code).Info("Resolving code")

	// L1: Redis cache
	url, err := s.cache.Get(ctx, code).Result()

	if err == nil && url != "" {
		logrus.WithField("code", code).Info("Cache hit")
		return &Dto{Code: code, OriginalURL: url}, nil
	}

	if err == redis.Nil || url == "" {
		logrus.WithField("code", code).Info("Cache miss")
	} else {
		logrus.WithError(err).WithField("code", code).Warn("Cache lookup error")
	}

	// L2: database
	link, err := s.repo.Find(ctx, code)

	if err != nil {
		if errors.Is(err, ErrNotFound) {
			logrus.WithField("code", code).Warn("Code not found in database")
		} else {
			logrus.WithError(err).WithField("code", code).Error("Database lookup error")
		}
		return nil, err
	}

	// Validate: must be active and not past its expiry.
	if !link.IsActive || (link.ExpiresAt != nil && link.ExpiresAt.Before(time.Now())) {
		logrus.WithField("code", code).Warn("Link is inactive or expired")
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
			logrus.WithError(err).WithField("code", code).Warn("Failed to send click event")
		}
	}()

	// Write-back to cache with a TTL that matches the link's remaining lifetime.
	if link.ExpiresAt != nil {
		if ttl := time.Until(*link.ExpiresAt); ttl > 0 {
			// Best-effort — a cache write failure must never break the redirect.
			if err := s.cache.Set(ctx, code, link.OriginalURL, ttl).Err(); err != nil {
				logrus.WithError(err).WithField("code", code).Warn("Failed to write to cache")
			} else {
				logrus.WithField("code", code).Info("Successfully wrote to cache")
			}
		}
	}

	return &Dto{
		Code:        link.Code,
		OriginalURL: link.OriginalURL,
	}, nil
}
