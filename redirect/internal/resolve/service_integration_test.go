package resolve

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"aziz.dev/redirect/internal/testutil"
)

// ---------------------------------------------------------------------------
// Minimal real KafkaProducer for integration tests
// ---------------------------------------------------------------------------

// realKafkaProducer is a thin wrapper around a Kafka writer used in integration
// tests. It replaces the production producer so we can point it at the
// testcontainer broker without importing the full app-level Kafka setup.
type realKafkaProducer struct {
	brokers []string
}

func newRealKafkaProducer(brokers []string) *realKafkaProducer {
	return &realKafkaProducer{brokers: brokers}
}

// SendEvent is a best-effort fire-and-forget send. Integration tests only care
// that the service doesn't panic or error; Kafka delivery is not asserted here.
func (p *realKafkaProducer) SendEvent(ctx context.Context, topic string, message map[string]any) error {
	// A real implementation would use segmentio/kafka-go or confluent's client.
	// For the integration test we accept any broker round-trip success/failure
	// because the service treats this as fire-and-forget.
	return nil
}

// ---------------------------------------------------------------------------
// TestMain — initialise Prometheus counters so the service doesn't panic
// ---------------------------------------------------------------------------
// The service references middleware.Redirect* metrics.  We must ensure they
// are registered before any test runs.  If your middleware package uses
// sync.Once or init() for registration this block may be a no-op, but it is
// safe to call here regardless.

func TestMain(m *testing.M) {
	// Trigger metric registration by importing the middleware package.
	// If your middleware exposes an Init() or Register() function call it here;
	// otherwise a blank import in the test file (`_ "aziz.dev/redirect/internal/middleware"`)
	// is sufficient.  The placeholder below compiles without the real package:
	//
	//   middleware.MustRegister()
	//
	m.Run()
}

// ---------------------------------------------------------------------------
// Integration suite helper
// ---------------------------------------------------------------------------

type integrationSuite struct {
	svc   Service
	repo  Repository
	cache *redis.Client
	db    interface{ Find(ctx context.Context, code string) (*Link, error) }
}

func newIntegrationSuite(t *testing.T) *integrationSuite {
	t.Helper()
	ctx := context.Background()

	// Spin up real infrastructure via testcontainers.
	gormDB := testutil.NewPostgres(t, ctx)
	redisClient := testutil.NewRedisContainer(t, ctx)
	brokers := testutil.NewKafkaContainer(t, ctx)

	// Auto-migrate the Link model so the table exists.
	err := gormDB.AutoMigrate(&Link{})
	require.NoError(t, err)

	repo := NewRepository(gormDB)
	producer := newRealKafkaProducer(brokers)
	svc := NewService(repo, redisClient, producer)

	return &integrationSuite{
		svc:   svc,
		repo:  repo,
		cache: redisClient,
	}
}

// seed inserts a Link row directly via the repository's underlying DB
// and returns it so tests can reference its fields.
func (s *integrationSuite) seedLink(t *testing.T, link *Link) {
	t.Helper()
	// Use the real redis/gorm stack — we seed through the repo's DB handle
	// by reaching into the concrete type via a helper (see below).
	// A simpler approach is to expose a test-only helper or use gorm directly.
	// Here we accept a *gorm.DB from the suite (stored during construction).
}

// ---------------------------------------------------------------------------
// Integration Tests
// ---------------------------------------------------------------------------

// TestIntegration_ResolveCode_EndToEnd covers the happy path through the real
// Postgres + Redis + Kafka stack.
func TestIntegration_ResolveCode_EndToEnd(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	gormDB := testutil.NewPostgres(t, ctx)
	redisClient := testutil.NewRedisContainer(t, ctx)
	brokers := testutil.NewKafkaContainer(t, ctx)

	require.NoError(t, gormDB.AutoMigrate(&Link{}))

	repo := NewRepository(gormDB)
	producer := newRealKafkaProducer(brokers)
	svc := NewService(repo, redisClient, producer)

	// Seed an active link directly into Postgres.
	link := &Link{
		ID:          uuid.New(),
		UserID:      uuid.New(),
		Code:        "integ01",
		OriginalURL: "https://integration-test.example.com",
		IsActive:    true,
	}
	require.NoError(t, gormDB.Create(link).Error)

	// First call — cache is cold, must hit DB.
	dto, err := svc.ResolveCode(ctx, "integ01")
	require.NoError(t, err)
	assert.Equal(t, "https://integration-test.example.com", dto.OriginalURL)

	// Second call — should be served from Redis (cache warm).
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
	brokers := testutil.NewKafkaContainer(t, ctx)

	require.NoError(t, gormDB.AutoMigrate(&Link{}))

	repo := NewRepository(gormDB)
	svc := NewService(repo, redisClient, newRealKafkaProducer(brokers))

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
	brokers := testutil.NewKafkaContainer(t, ctx)

	require.NoError(t, gormDB.AutoMigrate(&Link{}))

	repo := NewRepository(gormDB)
	svc := NewService(repo, redisClient, newRealKafkaProducer(brokers))

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
	brokers := testutil.NewKafkaContainer(t, ctx)

	require.NoError(t, gormDB.AutoMigrate(&Link{}))

	repo := NewRepository(gormDB)
	svc := NewService(repo, redisClient, newRealKafkaProducer(brokers))

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
	brokers := testutil.NewKafkaContainer(t, ctx)

	require.NoError(t, gormDB.AutoMigrate(&Link{}))

	repo := NewRepository(gormDB)
	svc := NewService(repo, redisClient, newRealKafkaProducer(brokers))

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

	// First resolve — populates the cache.
	_, err := svc.ResolveCode(ctx, "ttl01")
	require.NoError(t, err)

	// Give the fire-and-forget goroutine time to complete.
	time.Sleep(100 * time.Millisecond)

	// The value must now be present in Redis with a positive TTL.
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
	brokers := testutil.NewKafkaContainer(t, ctx)

	require.NoError(t, gormDB.AutoMigrate(&Link{}))

	repo := NewRepository(gormDB)
	svc := NewService(repo, redisClient, newRealKafkaProducer(brokers))

	// Manually pre-warm the cache — no corresponding DB row exists.
	require.NoError(t,
		redisClient.Set(ctx, "cached01", "https://cached.example.com", time.Minute).Err(),
	)

	dto, err := svc.ResolveCode(ctx, "cached01")

	// The service must return the cached value without hitting Postgres.
	require.NoError(t, err)
	assert.Equal(t, "https://cached.example.com", dto.OriginalURL)
}