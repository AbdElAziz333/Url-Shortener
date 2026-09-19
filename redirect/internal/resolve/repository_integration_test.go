package resolve

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	sqlcgen "aziz.dev/redirect/internal/postgres/sqlc"
	"aziz.dev/redirect/internal/testutil"
)

const linkTableSchema = `
	CREATE EXTENSION IF NOT EXISTS citext;

	CREATE TABLE IF NOT EXISTS link (
		id UUID PRIMARY KEY,
		user_id UUID NOT NULL,
		code VARCHAR(12) NOT NULL UNIQUE,
		original_url TEXT NOT NULL,
		custom_alias CITEXT UNIQUE,
		expires_at TIMESTAMPTZ,
		is_active BOOLEAN NOT NULL DEFAULT TRUE,
		created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
	);
`

const insertLinkSQL = `
	INSERT INTO link (
		id, user_id, code, original_url, custom_alias, expires_at, is_active, created_at
	) VALUES (
		$1, $2, $3, $4, $5, $6, $7, $8
	)
`

// ---------------------------------------------------------------------------
// Helper
// ---------------------------------------------------------------------------

func setupTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx := context.Background()
	db := testutil.NewPostgres(t, ctx)

	_, err := db.Exec(ctx, linkTableSchema)
	require.NoError(t, err)

	t.Cleanup(func() {
		_, _ = db.Exec(context.Background(), `DROP TABLE IF EXISTS link CASCADE;`)
	})

	return db
}

func newLink(code, url string) *sqlcgen.Link {
	return &sqlcgen.Link{
		ID:          uuid.New(),
		UserID:      uuid.New(),
		Code:        code,
		OriginalUrl: url,
		IsActive:    true,
	}
}

func seedLink(t *testing.T, ctx context.Context, db *pgxpool.Pool, link *sqlcgen.Link) {
	t.Helper()

	_, err := db.Exec(
		ctx,
		insertLinkSQL,
		link.ID,
		link.UserID,
		link.Code,
		link.OriginalUrl,
		link.CustomAlias,
		link.ExpiresAt,
		link.IsActive,
		link.CreatedAt,
	)

	require.NoError(t, err)
}

// ---------------------------------------------------------------------------
// Integration Tests
// ---------------------------------------------------------------------------

func TestRepository_Find_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	db := setupTestDB(t)
	// queries := sqlcgen.New(db)
	repo := NewRepository(db)

	link := newLink("abc123", "https://example.com")
	seedLink(t, ctx, db, link)

	found, err := repo.Find(ctx, "abc123")

	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, "abc123", found.Code)
	assert.Equal(t, "https://example.com", found.OriginalUrl)
	assert.True(t, found.IsActive)
}

func TestRepository_Find_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	db := setupTestDB(t)
	repo := NewRepository(db)

	_, err := repo.Find(ctx, "doesnotexist")

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestRepository_Find_ErrNotFound_WrapsCode(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	db := setupTestDB(t)
	repo := NewRepository(db)

	_, err := repo.Find(ctx, "xyz")

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotFound)
	// Error message should include the code for debuggability.
	assert.Contains(t, err.Error(), "xyz")
}

func TestRepository_Find_NilExpiresAt(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	db := testutil.NewPostgres(t, ctx)
	repo := NewRepository(db)

	link := newLink("noexpiry", "https://example.com")
	link.ExpiresAt = nil
	seedLink(t, ctx, db, link)

	found, err := repo.Find(ctx, "noexpiry")

	require.NoError(t, err)
	assert.Nil(t, found.ExpiresAt)
}

func TestRepository_Find_WithExpiresAt(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	db := testutil.NewPostgres(t, ctx)
	repo := NewRepository(db)

	future := time.Now().Add(24 * time.Hour).Truncate(time.Millisecond).UTC()
	link := newLink("withexpiry", "https://example.com")
	link.ExpiresAt = &future
	seedLink(t, ctx, db, link)

	found, err := repo.Find(ctx, "withexpiry")

	require.NoError(t, err)
	require.NotNil(t, found.ExpiresAt)
	assert.WithinDuration(t, future, *found.ExpiresAt, time.Millisecond)
}

func TestRepository_Find_UniqueCodeConstraint(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	db := testutil.NewPostgres(t, ctx)

	link1 := newLink("dupe", "https://first.example.com")
	seedLink(t, ctx, db, link1)

	// require.NoError(t, db.Create(link1).Error)

	link2 := newLink("dupe", "https://second.example.com")
	seedLink(t, ctx, db, link2)
	_, err := db.Exec(
		ctx, 
		insertLinkSQL,
		link2.UserID,
		link2.Code,
		link2.OriginalUrl,
		link2.CustomAlias,
		link2.ExpiresAt,
		link2.IsActive,
		link2.CreatedAt,
	)

	require.Error(t, err, "inserting a duplicate code should fail")
}

func TestRepository_Find_ReturnsCorrectRowByCode(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	db := testutil.NewPostgres(t, ctx)
	repo := NewRepository(db)

	seedLink(t, ctx, db, newLink("code-a", "https://a.example.com"))
	seedLink(t, ctx, db, newLink("code-b", "https://b.example.com"))
	seedLink(t, ctx, db, newLink("code-c", "https://c.example.com"))

	found, err := repo.Find(ctx, "code-b")

	require.NoError(t, err)
	assert.Equal(t, "https://b.example.com", found.OriginalUrl)
}