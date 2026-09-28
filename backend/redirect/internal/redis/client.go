package redis

import (
	"context"

	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"

	"aziz.dev/redirect/internal/config"
)

func NewClient(ctx context.Context, cfg *config.RedisConfig) (*redis.Client, error) {
	logrus.WithField("addr", cfg.Addr).Info("Connecting to Redis")
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Username: cfg.User,
		Password: cfg.Password,
	})
	
	err := client.Ping(ctx).Err()
	if err != nil {
		logrus.WithError(err).Error("Failed to connect to Redis")
		return nil, err
	}

	logrus.Info("Successfully connected to Redis")
	return client, nil
}