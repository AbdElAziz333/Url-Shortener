package testutil

import (
	"context"
	"testing"

	redisApi "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/redis"
)

func NewRedisContainer(t *testing.T, ctx context.Context) *redisApi.Client {
	t.Helper()

	redisContainer, err := redis.Run(ctx, "redis:8.6.2-alpine")
	require.NoError(t, err)

	addr, err := redisContainer.Endpoint(ctx, "")
	require.NoError(t, err)

	client := redisApi.NewClient(&redisApi.Options{
		Addr: addr,
	})

	t.Cleanup(func() {
		_ = client.Close()
		_ = redisContainer.Terminate(ctx)
	})
	
	err = client.Ping(ctx).Err()
	require.NoError(t, err)

	return client
}