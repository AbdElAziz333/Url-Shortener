package resolve

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"aziz.dev/redirect/internal/testutil"
)

// ---------------------------------------------------------------------------
// Helper
// ---------------------------------------------------------------------------

func newTestDB(t *testing.T) Repository {
	t.Helper()

	db := testutil.NewPostgres(t, context.Background())
	require.NoError(t, db.AutoMigrate(&Link{}))

	return NewRepository(db)
}

func newLink(code, url string) *Link {
	return &Link{
		ID:          uuid.New(),
		UserID:      uuid.New(),
		Code:        code,
		OriginalURL: url,
		IsActive:    true,
	}
}

// ---------------------------------------------------------------------------
// Integration Tests
// ---------------------------------------------------------------------------

func TestRepository_Find_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	db := testutil.NewPostgres(t, ctx)
	require.NoError(t, db.AutoMigrate(&Link{}))
	repo := NewRepository(db)

	link := newLink("abc123", "https://example.com")
	require.NoError(t, db.Create(link).Error)

	found, err := repo.Find(ctx, "abc123")

	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, "abc123", found.Code)
	assert.Equal(t, "https://example.com", found.OriginalURL)
	assert.True(t, found.IsActive)
}

func TestRepository_Find_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	repo := newTestDB(t)

	_, err := repo.Find(ctx, "doesnotexist")

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestRepository_Find_ErrNotFound_WrapsCode(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	repo := newTestDB(t)

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
	require.NoError(t, db.AutoMigrate(&Link{}))
	repo := NewRepository(db)

	link := newLink("noexpiry", "https://example.com")
	link.ExpiresAt = nil
	require.NoError(t, db.Create(link).Error)

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
	require.NoError(t, db.AutoMigrate(&Link{}))
	repo := NewRepository(db)

	future := time.Now().Add(24 * time.Hour).Truncate(time.Millisecond).UTC()
	link := newLink("withexpiry", "https://example.com")
	link.ExpiresAt = &future
	require.NoError(t, db.Create(link).Error)

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
	require.NoError(t, db.AutoMigrate(&Link{}))

	link1 := newLink("dupe", "https://first.example.com")
	require.NoError(t, db.Create(link1).Error)

	link2 := newLink("dupe", "https://second.example.com")
	err := db.Create(link2).Error

	require.Error(t, err, "inserting a duplicate code should fail")
}

func TestRepository_Find_ReturnsCorrectRowByCode(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	db := testutil.NewPostgres(t, ctx)
	require.NoError(t, db.AutoMigrate(&Link{}))
	repo := NewRepository(db)

	require.NoError(t, db.Create(newLink("code-a", "https://a.example.com")).Error)
	require.NoError(t, db.Create(newLink("code-b", "https://b.example.com")).Error)
	require.NoError(t, db.Create(newLink("code-c", "https://c.example.com")).Error)

	found, err := repo.Find(ctx, "code-b")

	require.NoError(t, err)
	assert.Equal(t, "https://b.example.com", found.OriginalURL)
}