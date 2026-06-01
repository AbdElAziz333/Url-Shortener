package resolve

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Mock Repository
// ---------------------------------------------------------------------------

type mockRepository struct {
	mock.Mock
}

func (m *mockRepository) Find(ctx context.Context, code string) (*Link, error) {
	args := m.Called(ctx, code)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Link), args.Error(1)
}

// ---------------------------------------------------------------------------
// Mock RedisClient
// ---------------------------------------------------------------------------

type mockRedisClient struct {
	mock.Mock
}

func (m *mockRedisClient) Get(ctx context.Context, key string) *redis.StringCmd {
	args := m.Called(ctx, key)
	return args.Get(0).(*redis.StringCmd)
}

func (m *mockRedisClient) Set(ctx context.Context, key string, value any, expiration time.Duration) *redis.StatusCmd {
	args := m.Called(ctx, key, value, expiration)
	return args.Get(0).(*redis.StatusCmd)
}

// ---------------------------------------------------------------------------
// Mock KafkaProducer
// ---------------------------------------------------------------------------

type mockKafkaProducer struct {
	mock.Mock
}

func (m *mockKafkaProducer) SendEvent(ctx context.Context, topic string, message map[string]any) error {
	args := m.Called(ctx, topic, message)
	return args.Error(0)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// cacheHit returns a StringCmd that simulates a Redis cache hit.
func cacheHit(url string) *redis.StringCmd {
	cmd := redis.NewStringCmd(context.Background())
	cmd.SetVal(url)
	return cmd
}

// cacheMiss returns a StringCmd that simulates a Redis cache miss (redis.Nil).
func cacheMiss() *redis.StringCmd {
	cmd := redis.NewStringCmd(context.Background())
	cmd.SetErr(redis.Nil)
	return cmd
}

// cacheErr returns a StringCmd that simulates a Redis transport error.
func cacheErr(err error) *redis.StringCmd {
	cmd := redis.NewStringCmd(context.Background())
	cmd.SetErr(err)
	return cmd
}

// okStatus returns a StatusCmd with no error (successful Set).
func okStatus() *redis.StatusCmd {
	cmd := redis.NewStatusCmd(context.Background())
	cmd.SetVal("OK")
	return cmd
}

// errStatus returns a StatusCmd that simulates a failed Set.
func errStatus(err error) *redis.StatusCmd {
	cmd := redis.NewStatusCmd(context.Background())
	cmd.SetErr(err)
	return cmd
}

func activeLink(code, url string) *Link {
	return &Link{
		Code:        code,
		OriginalURL: url,
		IsActive:    true,
		ExpiresAt:   nil,
	}
}

func expiredLink(code, url string) *Link {
	past := time.Now().Add(-time.Hour)
	return &Link{
		Code:        code,
		OriginalURL: url,
		IsActive:    true,
		ExpiresAt:   &past,
	}
}

func inactiveLink(code, url string) *Link {
	return &Link{
		Code:        code,
		OriginalURL: url,
		IsActive:    false,
	}
}

func futureTTLLink(code, url string, ttl time.Duration) *Link {
	future := time.Now().Add(ttl)
	return &Link{
		Code:        code,
		OriginalURL: url,
		IsActive:    true,
		ExpiresAt:   &future,
	}
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestService_ResolveCode_CacheHit(t *testing.T) {
	repo := new(mockRepository)
	cache := new(mockRedisClient)
	kafka := new(mockKafkaProducer)

	cache.On("Get", mock.Anything, "abc").Return(cacheHit("https://example.com"))
	// Repo and Kafka must NOT be called on a cache hit.

	svc := NewService(repo, cache, kafka)
	dto, err := svc.ResolveCode(context.Background(), "abc")

	require.NoError(t, err)
	assert.Equal(t, "https://example.com", dto.OriginalURL)
	assert.Equal(t, "abc", dto.Code)

	repo.AssertNotCalled(t, "Find", mock.Anything, mock.Anything)
	kafka.AssertNotCalled(t, "SendEvent", mock.Anything, mock.Anything, mock.Anything)
}

func TestService_ResolveCode_CacheMiss_DBHit(t *testing.T) {
	repo := new(mockRepository)
	cache := new(mockRedisClient)
	kafka := new(mockKafkaProducer)

	link := activeLink("abc", "https://example.com")
	cache.On("Get", mock.Anything, "abc").Return(cacheMiss())
	repo.On("Find", mock.Anything, "abc").Return(link, nil)
	// No ExpiresAt → no cache Set call expected.
	kafka.On("SendEvent", mock.Anything, "url-clicked", mock.Anything).Return(nil)

	svc := NewService(repo, cache, kafka)
	dto, err := svc.ResolveCode(context.Background(), "abc")

	require.NoError(t, err)
	assert.Equal(t, "https://example.com", dto.OriginalURL)
	repo.AssertExpectations(t)
}

func TestService_ResolveCode_CacheMiss_DBHit_WithTTL_WritesBackToCache(t *testing.T) {
	repo := new(mockRepository)
	cache := new(mockRedisClient)
	kafka := new(mockKafkaProducer)

	ttl := 10 * time.Minute
	link := futureTTLLink("abc", "https://example.com", ttl)

	cache.On("Get", mock.Anything, "abc").Return(cacheMiss())
	repo.On("Find", mock.Anything, "abc").Return(link, nil)
	// Cache write-back: TTL must be positive and roughly match the remaining lifetime.
	cache.On("Set", mock.Anything, "abc", "https://example.com", mock.MatchedBy(func(d time.Duration) bool {
		return d > 0 && d <= ttl
	})).Return(okStatus())
	kafka.On("SendEvent", mock.Anything, "url-clicked", mock.Anything).Return(nil)

	svc := NewService(repo, cache, kafka)
	dto, err := svc.ResolveCode(context.Background(), "abc")

	require.NoError(t, err)
	assert.Equal(t, "https://example.com", dto.OriginalURL)
	// Allow the fire-and-forget goroutine to finish before asserting Kafka.
	time.Sleep(50 * time.Millisecond)
	cache.AssertExpectations(t)
	kafka.AssertExpectations(t)
}

func TestService_ResolveCode_CacheWriteFailure_DoesNotBreakRedirect(t *testing.T) {
	repo := new(mockRepository)
	cache := new(mockRedisClient)
	kafka := new(mockKafkaProducer)

	ttl := 5 * time.Minute
	link := futureTTLLink("abc", "https://example.com", ttl)

	cache.On("Get", mock.Anything, "abc").Return(cacheMiss())
	repo.On("Find", mock.Anything, "abc").Return(link, nil)
	cache.On("Set", mock.Anything, "abc", "https://example.com", mock.Anything).
		Return(errStatus(errors.New("redis timeout")))
	kafka.On("SendEvent", mock.Anything, "url-clicked", mock.Anything).Return(nil)

	svc := NewService(repo, cache, kafka)
	dto, err := svc.ResolveCode(context.Background(), "abc")

	// A cache write failure must never surface as an error to the caller.
	require.NoError(t, err)
	assert.Equal(t, "https://example.com", dto.OriginalURL)
}

func TestService_ResolveCode_DBNotFound(t *testing.T) {
	repo := new(mockRepository)
	cache := new(mockRedisClient)
	kafka := new(mockKafkaProducer)

	cache.On("Get", mock.Anything, "missing").Return(cacheMiss())
	repo.On("Find", mock.Anything, "missing").Return(nil, ErrNotFound)

	svc := NewService(repo, cache, kafka)
	dto, err := svc.ResolveCode(context.Background(), "missing")

	require.Error(t, err)
	assert.Nil(t, dto)
	assert.True(t, errors.Is(err, ErrNotFound))
}

func TestService_ResolveCode_InactiveLink(t *testing.T) {
	repo := new(mockRepository)
	cache := new(mockRedisClient)
	kafka := new(mockKafkaProducer)

	link := inactiveLink("abc", "https://example.com")
	cache.On("Get", mock.Anything, "abc").Return(cacheMiss())
	repo.On("Find", mock.Anything, "abc").Return(link, nil)

	svc := NewService(repo, cache, kafka)
	dto, err := svc.ResolveCode(context.Background(), "abc")

	require.Error(t, err)
	assert.Nil(t, dto)
	assert.ErrorIs(t, err, ErrLinkInactive)
	kafka.AssertNotCalled(t, "SendEvent", mock.Anything, mock.Anything, mock.Anything)
}

func TestService_ResolveCode_ExpiredLink(t *testing.T) {
	repo := new(mockRepository)
	cache := new(mockRedisClient)
	kafka := new(mockKafkaProducer)

	link := expiredLink("abc", "https://example.com")
	cache.On("Get", mock.Anything, "abc").Return(cacheMiss())
	repo.On("Find", mock.Anything, "abc").Return(link, nil)

	svc := NewService(repo, cache, kafka)
	dto, err := svc.ResolveCode(context.Background(), "abc")

	require.Error(t, err)
	assert.Nil(t, dto)
	assert.ErrorIs(t, err, ErrLinkInactive)
	kafka.AssertNotCalled(t, "SendEvent", mock.Anything, mock.Anything, mock.Anything)
}

func TestService_ResolveCode_KafkaFailure_DoesNotBreakRedirect(t *testing.T) {
	repo := new(mockRepository)
	cache := new(mockRedisClient)
	kafka := new(mockKafkaProducer)

	link := activeLink("abc", "https://example.com")
	cache.On("Get", mock.Anything, "abc").Return(cacheMiss())
	repo.On("Find", mock.Anything, "abc").Return(link, nil)
	kafka.On("SendEvent", mock.Anything, "url-clicked", mock.Anything).
		Return(errors.New("broker unavailable"))

	svc := NewService(repo, cache, kafka)
	dto, err := svc.ResolveCode(context.Background(), "abc")

	// Kafka runs in a fire-and-forget goroutine — its failure must not propagate.
	require.NoError(t, err)
	assert.Equal(t, "https://example.com", dto.OriginalURL)

	time.Sleep(50 * time.Millisecond)
	kafka.AssertExpectations(t)
}

func TestService_ResolveCode_CacheTransportError_FallsThrough(t *testing.T) {
	repo := new(mockRepository)
	cache := new(mockRedisClient)
	kafka := new(mockKafkaProducer)

	link := activeLink("abc", "https://example.com")
	cache.On("Get", mock.Anything, "abc").Return(cacheErr(errors.New("connection refused")))
	repo.On("Find", mock.Anything, "abc").Return(link, nil)
	kafka.On("SendEvent", mock.Anything, "url-clicked", mock.Anything).Return(nil)

	svc := NewService(repo, cache, kafka)
	dto, err := svc.ResolveCode(context.Background(), "abc")

	// A Redis transport error is not a hit — service must fall through to the DB.
	require.NoError(t, err)
	assert.Equal(t, "https://example.com", dto.OriginalURL)
	repo.AssertExpectations(t)
}

func TestService_ResolveCode_KafkaPayloadContainsExpectedFields(t *testing.T) {
	repo := new(mockRepository)
	cache := new(mockRedisClient)
	kafka := new(mockKafkaProducer)

	link := activeLink("abc", "https://example.com")
	cache.On("Get", mock.Anything, "abc").Return(cacheMiss())
	repo.On("Find", mock.Anything, "abc").Return(link, nil)
	kafka.On("SendEvent", mock.Anything, "url-clicked", mock.MatchedBy(func(m map[string]any) bool {
		_, hasCode := m["code"]
		_, hasURL := m["original_url"]
		_, hasClickedAt := m["clicked_at"]
		return hasCode && hasURL && hasClickedAt
	})).Return(nil)

	svc := NewService(repo, cache, kafka)
	_, err := svc.ResolveCode(context.Background(), "abc")

	require.NoError(t, err)
	time.Sleep(50 * time.Millisecond)
	kafka.AssertExpectations(t)
}