// go:build integration
// +build integration
//
// Run with: go test -tags=integration ./...
//
// These tests spin up real Postgres and MongoDB containers via testcontainers-go.
// They verify the full repository behaviour end-to-end, including SQL upserts
// and MongoDB aggregation pipelines.

package stat

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/mongodb"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	mongoDriver "go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	gormPostgres "gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func startPostgres(t *testing.T, ctx context.Context) *gorm.DB {
	t.Helper()

	pgCtr, err := postgres.RunContainer(ctx,
		testcontainers.WithImage("postgres:16-alpine"),
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = pgCtr.Terminate(ctx) })

	dsn, err := pgCtr.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	db, err := gorm.Open(gormPostgres.Open(dsn), &gorm.Config{})
	require.NoError(t, err)

	// Create the table used by the repository.
	require.NoError(t, db.Exec(`
		CREATE TABLE IF NOT EXISTS link_stats_daily (
			id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			link_code   TEXT NOT NULL,
			date        DATE NOT NULL,
			click_count BIGINT NOT NULL DEFAULT 0,
			UNIQUE (link_code, date)
		)
	`).Error)

	return db
}

func startMongo(t *testing.T, ctx context.Context) *mongoDriver.Client {
	t.Helper()

	mongoCtr, err := mongodb.RunContainer(ctx,
		testcontainers.WithImage("mongo:7"),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = mongoCtr.Terminate(ctx) })

	uri, err := mongoCtr.ConnectionString(ctx)
	require.NoError(t, err)

	client, err := mongoDriver.Connect(options.Client().ApplyURI(uri))
	require.NoError(t, err)
	t.Cleanup(func() { _ = client.Disconnect(ctx) })

	return client
}

// ---------------------------------------------------------------------------
// SaveClickEvent
// ---------------------------------------------------------------------------

func TestIntegration_SaveClickEvent_InsertsMongoAndPostgres(t *testing.T) {
	ctx := context.Background()
	db := startPostgres(t, ctx)
	mc := startMongo(t, ctx)

	repo := NewRepository(db, mc)

	event := map[string]any{
		"code":    "integ1",
		"country": "US",
		"referrer_domain": "google.com",
	}
	require.NoError(t, repo.SaveClickEvent(ctx, event))

	// Verify Postgres upsert.
	var count int64
	require.NoError(t, db.Raw(
		"SELECT click_count FROM link_stats_daily WHERE link_code = ?", "integ1",
	).Scan(&count).Error)
	assert.Equal(t, int64(1), count)

	// Verify Mongo insert.
	coll := mc.Database("analytics").Collection("click_events")
	n, err := coll.CountDocuments(ctx, map[string]any{"code": "integ1"})
	require.NoError(t, err)
	assert.Equal(t, int64(1), n)
}

func TestIntegration_SaveClickEvent_UpsertIncrementsClickCount(t *testing.T) {
	ctx := context.Background()
	db := startPostgres(t, ctx)
	mc := startMongo(t, ctx)

	repo := NewRepository(db, mc)

	event := map[string]any{"code": "upsert1"}

	require.NoError(t, repo.SaveClickEvent(ctx, event))
	require.NoError(t, repo.SaveClickEvent(ctx, event))
	require.NoError(t, repo.SaveClickEvent(ctx, event))

	var count int64
	require.NoError(t, db.Raw(
		"SELECT click_count FROM link_stats_daily WHERE link_code = ?", "upsert1",
	).Scan(&count).Error)
	assert.Equal(t, int64(3), count)
}

func TestIntegration_SaveClickEvent_MissingCodeSkipsPostgres(t *testing.T) {
	ctx := context.Background()
	db := startPostgres(t, ctx)
	mc := startMongo(t, ctx)

	repo := NewRepository(db, mc)

	// Event without a "code" key — should not write to link_stats_daily.
	event := map[string]any{"country": "DE"}
	require.NoError(t, repo.SaveClickEvent(ctx, event))

	var count int64
	db.Raw("SELECT COUNT(*) FROM link_stats_daily").Scan(&count)
	assert.Equal(t, int64(0), count)
}

// ---------------------------------------------------------------------------
// GetTotalClicks
// ---------------------------------------------------------------------------

func TestIntegration_GetTotalClicks_ReturnsSum(t *testing.T) {
	ctx := context.Background()
	db := startPostgres(t, ctx)
	mc := startMongo(t, ctx)

	repo := NewRepository(db, mc)

	// Insert two separate days.
	require.NoError(t, db.Exec(`
		INSERT INTO link_stats_daily (link_code, date, click_count)
		VALUES ('clicks1', CURRENT_DATE, 10),
		       ('clicks1', CURRENT_DATE - 1, 5)
	`).Error)

	result, err := repo.GetTotalClicks(ctx, "clicks1")
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, "total", result[0].Key)
	assert.Equal(t, int64(15), result[0].Count)
}

func TestIntegration_GetTotalClicks_NoRows_ReturnsZero(t *testing.T) {
	ctx := context.Background()
	db := startPostgres(t, ctx)
	mc := startMongo(t, ctx)

	repo := NewRepository(db, mc)

	result, err := repo.GetTotalClicks(ctx, "nonexistent")
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, int64(0), result[0].Count)
}

func TestIntegration_GetTotalClicks_IsolatedByCode(t *testing.T) {
	ctx := context.Background()
	db := startPostgres(t, ctx)
	mc := startMongo(t, ctx)

	repo := NewRepository(db, mc)

	require.NoError(t, db.Exec(`
		INSERT INTO link_stats_daily (link_code, date, click_count)
		VALUES ('codeA', CURRENT_DATE, 100),
		       ('codeB', CURRENT_DATE, 999)
	`).Error)

	result, err := repo.GetTotalClicks(ctx, "codeA")
	require.NoError(t, err)
	assert.Equal(t, int64(100), result[0].Count)
}

// ---------------------------------------------------------------------------
// GetGeo
// ---------------------------------------------------------------------------

func TestIntegration_GetGeo_ReturnsTopCountries(t *testing.T) {
	ctx := context.Background()
	db := startPostgres(t, ctx)
	mc := startMongo(t, ctx)
	repo := NewRepository(db, mc)

	coll := mc.Database("analytics").Collection("click_events")
	docs := []any{
		map[string]any{"code": "geo1", "country": "US"},
		map[string]any{"code": "geo1", "country": "US"},
		map[string]any{"code": "geo1", "country": "DE"},
		map[string]any{"code": "other", "country": "FR"}, // different code
	}
	_, err := coll.InsertMany(ctx, docs)
	require.NoError(t, err)

	result, err := repo.GetGeo(ctx, "geo1")
	require.NoError(t, err)

	// Results are sorted descending by count.
	require.Len(t, result, 2)
	assert.Equal(t, "US", result[0].Key)
	assert.Equal(t, int64(2), result[0].Count)
	assert.Equal(t, "DE", result[1].Key)
	assert.Equal(t, int64(1), result[1].Count)
}

func TestIntegration_GetGeo_NullCountryFallsBackToUnknown(t *testing.T) {
	ctx := context.Background()
	db := startPostgres(t, ctx)
	mc := startMongo(t, ctx)
	repo := NewRepository(db, mc)

	coll := mc.Database("analytics").Collection("click_events")
	_, err := coll.InsertMany(ctx, []any{
		map[string]any{"code": "geo2"},           // no country field
		map[string]any{"code": "geo2", "country": nil}, // explicit null
	})
	require.NoError(t, err)

	result, err := repo.GetGeo(ctx, "geo2")
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, "unknown", result[0].Key)
	assert.Equal(t, int64(2), result[0].Count)
}

func TestIntegration_GetGeo_LimitTopTen(t *testing.T) {
	ctx := context.Background()
	db := startPostgres(t, ctx)
	mc := startMongo(t, ctx)
	repo := NewRepository(db, mc)

	coll := mc.Database("analytics").Collection("click_events")
	docs := make([]any, 15)
	for i := range docs {
		docs[i] = map[string]any{
			"code":    "geoLimit",
			"country": fmt.Sprintf("C%d", i),
		}
	}
	_, err := coll.InsertMany(ctx, docs)
	require.NoError(t, err)

	result, err := repo.GetGeo(ctx, "geoLimit")
	require.NoError(t, err)
	assert.LessOrEqual(t, len(result), 10)
}

func TestIntegration_GetGeo_NoEvents_ReturnsEmpty(t *testing.T) {
	ctx := context.Background()
	db := startPostgres(t, ctx)
	mc := startMongo(t, ctx)
	repo := NewRepository(db, mc)

	result, err := repo.GetGeo(ctx, "nosuchcode")
	require.NoError(t, err)
	assert.Empty(t, result)
}

// ---------------------------------------------------------------------------
// GetReferrers
// ---------------------------------------------------------------------------

func TestIntegration_GetReferrers_ReturnsTopDomains(t *testing.T) {
	ctx := context.Background()
	db := startPostgres(t, ctx)
	mc := startMongo(t, ctx)
	repo := NewRepository(db, mc)

	coll := mc.Database("analytics").Collection("click_events")
	_, err := coll.InsertMany(ctx, []any{
		map[string]any{"code": "ref1", "referrer_domain": "google.com"},
		map[string]any{"code": "ref1", "referrer_domain": "google.com"},
		map[string]any{"code": "ref1", "referrer_domain": "twitter.com"},
		map[string]any{"code": "other", "referrer_domain": "bing.com"},
	})
	require.NoError(t, err)

	result, err := repo.GetReferrers(ctx, "ref1")
	require.NoError(t, err)
	require.Len(t, result, 2)
	assert.Equal(t, "google.com", result[0].Key)
	assert.Equal(t, int64(2), result[0].Count)
}

func TestIntegration_GetReferrers_FallsBackToReferrerField(t *testing.T) {
	ctx := context.Background()
	db := startPostgres(t, ctx)
	mc := startMongo(t, ctx)
	repo := NewRepository(db, mc)

	coll := mc.Database("analytics").Collection("click_events")
	_, err := coll.InsertMany(ctx, []any{
		// No referrer_domain; falls back to referrer field.
		map[string]any{"code": "ref2", "referrer": "https://example.com/page"},
	})
	require.NoError(t, err)

	result, err := repo.GetReferrers(ctx, "ref2")
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, "https://example.com/page", result[0].Key)
}

func TestIntegration_GetReferrers_NullReferrerFallsBackToUnknown(t *testing.T) {
	ctx := context.Background()
	db := startPostgres(t, ctx)
	mc := startMongo(t, ctx)
	repo := NewRepository(db, mc)

	coll := mc.Database("analytics").Collection("click_events")
	_, err := coll.InsertMany(ctx, []any{
		map[string]any{"code": "ref3"}, // no referrer fields at all
	})
	require.NoError(t, err)

	result, err := repo.GetReferrers(ctx, "ref3")
	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, "unknown", result[0].Key)
}

func TestIntegration_GetReferrers_LimitTopTen(t *testing.T) {
	ctx := context.Background()
	db := startPostgres(t, ctx)
	mc := startMongo(t, ctx)
	repo := NewRepository(db, mc)

	coll := mc.Database("analytics").Collection("click_events")
	docs := make([]any, 15)
	for i := range docs {
		docs[i] = map[string]any{
			"code":            "refLimit",
			"referrer_domain": fmt.Sprintf("site%d.com", i),
		}
	}
	_, err := coll.InsertMany(ctx, docs)
	require.NoError(t, err)

	result, err := repo.GetReferrers(ctx, "refLimit")
	require.NoError(t, err)
	assert.LessOrEqual(t, len(result), 10)
}