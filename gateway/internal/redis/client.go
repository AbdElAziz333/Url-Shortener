package redis

import (
	"context"

	"github.com/redis/go-redis/v9"

	"aziz.dev/gateway/internal/config"
)

func NewClient(ctx context.Context, cfg *config.RedisConfig) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr: cfg.Addr,
		Username: cfg.Username,
		Password: cfg.Password,
	})
	
	err := client.Ping(ctx).Err()
	if err != nil {
		return nil, err
	}

	return client, nil
}