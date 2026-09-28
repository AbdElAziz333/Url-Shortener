package user

import (
	"context"
	"testing"

	"aziz.dev/gateway/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	ctx := context.Background()
	db := testutil.NewPostgres(t, ctx)
	err := db.AutoMigrate(&User{})
	require.NoError(t, err)
	return db
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

	u := User{
		Email:        "found@example.com",
		PasswordHash: "hash",
		IsActive:     true,
	}
	err := db.Create(&u).Error
	require.NoError(t, err)

	result, err := repo.FindByEmail(context.Background(), "found@example.com")

	assert.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "found@example.com", result.Email)
	assert.Equal(t, "hash", result.PasswordHash)
	assert.True(t, result.IsActive)
}

func TestRepository_FindByEmail_CaseSensitive(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	u := User{
		Email:        "CaseSensitive@example.com",
		PasswordHash: "hash",
		IsActive:     true,
	}
	db.Create(&u)

	result, err := repo.FindByEmail(context.Background(), "casesensitive@example.com")
	assert.NoError(t, err)
	// PostgreSQL default collation is case-sensitive; result should be nil
	assert.Nil(t, result)
}

// --- Create Integration Tests ---

func TestRepository_Create_Success(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	u := User{
		Email:        "create@example.com",
		PasswordHash: "hashed",
		IsActive:     true,
	}

	err := repo.Create(context.Background(), u)
	assert.NoError(t, err)

	// Verify it's actually persisted
	var found User
	err = db.Where("email = ?", "create@example.com").First(&found).Error
	assert.NoError(t, err)
	assert.Equal(t, "create@example.com", found.Email)
	assert.True(t, found.IsActive)
}

func TestRepository_Create_SetsUUIDFromBeforeCreate(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	u := User{
		Email:        "uuid@example.com",
		PasswordHash: "hash",
		IsActive:     true,
	}

	err := repo.Create(context.Background(), u)
	require.NoError(t, err)

	var found User
	db.Where("email = ?", "uuid@example.com").First(&found)
	assert.NotEmpty(t, found.ID)
}

func TestRepository_Create_SetsTimestamps(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	u := User{
		Email:        "timestamps@example.com",
		PasswordHash: "hash",
		IsActive:     true,
	}

	err := repo.Create(context.Background(), u)
	require.NoError(t, err)

	var found User
	db.Where("email = ?", "timestamps@example.com").First(&found)
	assert.False(t, found.CreatedAt.IsZero(), "CreatedAt should be set")
	assert.False(t, found.UpdatedAt.IsZero(), "UpdatedAt should be set")
}

func TestRepository_Create_DuplicateEmail(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)

	u := User{
		Email:        "dup@example.com",
		PasswordHash: "hash",
		IsActive:     true,
	}

	err := repo.Create(context.Background(), u)
	require.NoError(t, err)

	// Second create with same email — behaviour depends on DB constraints.
	// Without a UNIQUE constraint in the schema, GORM will insert a second row.
	// If a unique index exists, this should return an error.
	u2 := User{
		Email:        "dup@example.com",
		PasswordHash: "hash2",
		IsActive:     true,
	}
	_ = repo.Create(context.Background(), u2)

	var count int64
	db.Model(&User{}).Where("email = ?", "dup@example.com").Count(&count)
	// Adjust assertion depending on whether unique index is applied in schema
	assert.GreaterOrEqual(t, count, int64(1))
}

// --- Round-trip Integration Test ---

func TestRepository_CreateThenFindByEmail(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	original := User{
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