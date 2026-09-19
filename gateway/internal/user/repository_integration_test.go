package user

import (
	"context"
	"testing"
	"time"

	sqlcgen "aziz.dev/gateway/internal/postgres/sqlc"
	"aziz.dev/gateway/internal/testutil"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const userTableSchema = `
	CREATE EXTENSION IF NOT EXISTS citext;

	CREATE TABLE IF NOT EXISTS users (
		id UUID NOT NULL PRIMARY KEY,
		email CITEXT NOT NULL UNIQUE,
		password_hash VARCHAR(255),
		is_active BOOLEAN NOT NULL DEFAULT TRUE,
		created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
	);
`

func setupTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx := context.Background()
	db := testutil.NewPostgres(t, ctx)

	_, err := db.Exec(ctx, userTableSchema)
	require.NoError(t, err)

	t.Cleanup(func() {
		_, _ = db.Exec(context.Background(), `DROP TABLE IF EXISTS users CASCADE;`)
	})

	return db
}

func insertUser(t *testing.T, db *pgxpool.Pool, u sqlcgen.User) {
	t.Helper()
	_, err := db.Exec(
		context.Background(),
		`INSERT INTO "user" (id, email, password_hash, is_active, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		u.ID, u.Email, u.PasswordHash, u.IsActive, u.CreatedAt, u.UpdatedAt,
	)
	require.NoError(t, err)
}

func newTestUser(email, passwordHash string) sqlcgen.User {
	now := time.Now().UTC()
	return sqlcgen.User{
		ID:           uuid.New(),
		Email:        email,
		PasswordHash: passwordHash,
		IsActive:     true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

// --- FindByEmail Integration Tests ---

func TestRepository_FindByEmail_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	result, err := repo.FindByEmail(context.Background(), "nobody@example.com")

	assert.NoError(t, err)
	assert.Nil(t, result)
}

func TestRepository_FindByEmail_Found(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	u := newTestUser("found@example.com", "hash")
	insertUser(t, db, u)

	result, err := repo.FindByEmail(context.Background(), "found@example.com")

	assert.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "found@example.com", result.Email)
	assert.Equal(t, "hash", result.PasswordHash)
	assert.True(t, result.IsActive)
}

func TestRepository_FindByEmail_CaseInsensitive(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	u := newTestUser("CaseSensitive@example.com", "hash")
	insertUser(t, db, u)

	result, err := repo.FindByEmail(context.Background(), "casesensitive@example.com")
	assert.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "CaseSensitive@example.com", result.Email)
}

// --- Create Integration Tests ---

func TestRepository_Create_Success(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	u := sqlcgen.User{
		Email:        "create@example.com",
		PasswordHash: "hashed",
		IsActive:     true,
	}

	err := repo.Create(ctx, u)
	assert.NoError(t, err)

	var found sqlcgen.User
	err = db.QueryRow(ctx, `
		SELECT id, email, password_hash, is_active, created_at, updated_at
		FROM "user"
		WHERE email = $1`, "create@example.com").Scan(
		&found.ID,
		&found.Email,
		&found.PasswordHash,
		&found.IsActive,
		&found.CreatedAt,
		&found.UpdatedAt,
	)

	assert.NoError(t, err)
	assert.Equal(t, "create@example.com", found.Email)
	assert.True(t, found.IsActive)
}

func TestRepository_Create_SetsUUIDFromBeforeCreate(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	u := sqlcgen.User{
		Email:        "uuid@example.com",
		PasswordHash: "hash",
		IsActive:     true,
	}

	err := repo.Create(ctx, u)
	require.NoError(t, err)

	var foundID uuid.UUID
	err = db.QueryRow(ctx, `SELECT id FROM "user" WHERE email = $1`, u.Email).Scan(&foundID)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, foundID)
}

func TestRepository_Create_SetsTimestamps(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	u := sqlcgen.User{
		Email:        "timestamps@example.com",
		PasswordHash: "hash",
		IsActive:     true,
	}

	err := repo.Create(ctx, u)
	require.NoError(t, err)

	var createdAt, updatedAt time.Time
	err = db.QueryRow(ctx, `SELECT created_at, updated_at FROM "user" WHERE email = $1`, "timestamps@example.com").Scan(&createdAt, &updatedAt)
	require.NoError(t, err)

	assert.False(t, createdAt.IsZero(), "CreatedAt should be set")
	assert.False(t, updatedAt.IsZero(), "UpdatedAt should be set")
}

func TestRepository_Create_DuplicateEmail(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	u := newTestUser("dup@example.com", "hash")
	err := repo.Create(ctx, u)
	require.NoError(t, err)

	u2 := newTestUser("dup@example.com", "hash2")
	err = repo.Create(ctx, u2)
	assert.Error(t, err, "expected error on unique constraint violation")

	var count int64
	err = db.QueryRow(ctx, `SELECT COUNT(*) FROM "user" WHERE email = $1`, "dup@example.com").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
}

// --- Round-trip Integration Test ---

func TestRepository_CreateThenFindByEmail(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	original := sqlcgen.User{
		Email:        "roundtrip@example.com",
		PasswordHash: "secureHash",
		IsActive:     true,
	}

	err := repo.Create(ctx, original)
	require.NoError(t, err)

	found, err := repo.FindByEmail(ctx, "roundtrip@example.com")
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, original.Email, found.Email)
	assert.Equal(t, original.PasswordHash, found.PasswordHash)
	assert.True(t, found.IsActive)
}
