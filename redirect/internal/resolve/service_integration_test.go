package resolve

import (
	"context"
	"errors"
	"testing"
	"time"

	"aziz.dev/redirect/internal/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegration_ResolveCode_EndToEnd(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	gormDB := testutil.NewPostgres(t, ctx)
	redisClient := testutil.NewRedisContainer(t, ctx)

	require.NoError(t, gormDB.AutoMigrate(&Link{}))

	repo := NewRepository(gormDB)
	svc := NewService(repo, redisClient)

	link := &Link{
		ID:          uuid.New(),
		UserID:      uuid.New(),
		Code:        "integ01",
		OriginalURL: "https://integration-test.example.com",
		IsActive:    true,
	}
	require.NoError(t, gormDB.Create(link).Error)

	dto, err := svc.ResolveCode(ctx, "integ01")
	require.NoError(t, err)
	assert.Equal(t, "https://integration-test.example.com", dto.OriginalURL)

	dto2, err := svc.ResolveCode(ctx, "integ01")
	require.NoError(t, err)
	assert.Equal(t, dto.OriginalURL, dto2.OriginalURL)
}

func TestIntegration_ResolveCode_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	gormDB := testutil.NewPostgres(t, ctx)
	redisClient := testutil.NewRedisContainer(t, ctx)

	require.NoError(t, gormDB.AutoMigrate(&Link{}))

	repo := NewRepository(gormDB)
	svc := NewService(repo, redisClient)

	_, err := svc.ResolveCode(ctx, "doesnotexist")

	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrNotFound))
}

func TestIntegration_ResolveCode_InactiveLink(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	gormDB := testutil.NewPostgres(t, ctx)
	redisClient := testutil.NewRedisContainer(t, ctx)

	require.NoError(t, gormDB.AutoMigrate(&Link{}))

	repo := NewRepository(gormDB)
	svc := NewService(repo, redisClient)

	link := &Link{
		ID:          uuid.New(),
		UserID:      uuid.New(),
		Code:        "inactive01",
		OriginalURL: "https://inactive.example.com",
		IsActive:    false,
	}
	require.NoError(t, gormDB.Create(link).Error)

	_, err := svc.ResolveCode(ctx, "inactive01")

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrLinkInactive)
}

func TestIntegration_ResolveCode_ExpiredLink(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	gormDB := testutil.NewPostgres(t, ctx)
	redisClient := testutil.NewRedisContainer(t, ctx)

	require.NoError(t, gormDB.AutoMigrate(&Link{}))

	repo := NewRepository(gormDB)
	svc := NewService(repo, redisClient)

	past := time.Now().Add(-1 * time.Hour)
	link := &Link{
		ID:          uuid.New(),
		UserID:      uuid.New(),
		Code:        "expired01",
		OriginalURL: "https://expired.example.com",
		IsActive:    true,
		ExpiresAt:   &past,
	}
	require.NoError(t, gormDB.Create(link).Error)

	_, err := svc.ResolveCode(ctx, "expired01")

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrLinkInactive)
}

func TestIntegration_ResolveCode_CacheWriteBack_WithTTL(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	gormDB := testutil.NewPostgres(t, ctx)
	redisClient := testutil.NewRedisContainer(t, ctx)

	require.NoError(t, gormDB.AutoMigrate(&Link{}))

	repo := NewRepository(gormDB)
	svc := NewService(repo, redisClient)

	future := time.Now().Add(30 * time.Minute)
	link := &Link{
		ID:          uuid.New(),
		UserID:      uuid.New(),
		Code:        "ttl01",
		OriginalURL: "https://ttl.example.com",
		IsActive:    true,
		ExpiresAt:   &future,
	}
	require.NoError(t, gormDB.Create(link).Error)

	_, err := svc.ResolveCode(ctx, "ttl01")
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	val, err := redisClient.Get(ctx, "ttl01").Result()
	require.NoError(t, err)
	assert.Equal(t, "https://ttl.example.com", val)

	ttl, err := redisClient.TTL(ctx, "ttl01").Result()
	require.NoError(t, err)
	assert.Positive(t, ttl, "cached key must have a positive TTL")
}

func TestIntegration_ResolveCode_CacheHit_DoesNotQueryDB(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	gormDB := testutil.NewPostgres(t, ctx)
	redisClient := testutil.NewRedisContainer(t, ctx)

	require.NoError(t, gormDB.AutoMigrate(&Link{}))

	repo := NewRepository(gormDB)
	svc := NewService(repo, redisClient)

	require.NoError(t,
		redisClient.Set(ctx, "cached01", "https://cached.example.com", time.Minute).Err(),
	)

	dto, err := svc.ResolveCode(ctx, "cached01")

	require.NoError(t, err)
	assert.Equal(t, "https://cached.example.com", dto.OriginalURL)
}
