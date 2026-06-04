package redis

import (
	"context"

	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"

	"aziz.dev/gateway/internal/config"
)

func NewClient(ctx context.Context, cfg *config.RedisConfig) (*redis.Client, error) {
	logrus.WithField("addr", cfg.Addr).Info("Initializing Redis client")
	client := redis.NewClient(&redis.Options{
		Addr: cfg.Addr,
		Username: cfg.Username,
		Password: cfg.Password,
	})
	
	err := client.Ping(ctx).Err()
	if err != nil {
		logrus.WithError(err).Error("Failed to ping Redis")
		return nil, err
	}

	logrus.Info("Redis client initialized")
	return client, nil
}