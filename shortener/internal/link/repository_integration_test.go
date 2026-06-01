package link

import (
	"context"
	"testing"
	"time"

	"aziz.dev/shortener/internal/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// setupTestDB provisions a real Postgres container, runs AutoMigrate, and
// registers cleanup via t.Cleanup — no manual teardown needed in tests.
func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	ctx := context.Background()
	db := testutil.NewPostgres(t, ctx)
	require.NoError(t, db.AutoMigrate(&Link{}))
	return db
}

func makeLink(userID uuid.UUID, code string) *Link {
	id, _ := uuid.NewV7()
	return &Link{
		ID:          id,
		UserID:      userID,
		Code:        code,
		OriginalURL: "https://example.com/" + code,
		IsActive:    true,
		CreatedAt:   time.Now(),
	}
}

// --- FindAllByUserID ---

func TestRepo_FindAllByUserID_ReturnsOnlyActiveLinksForUser(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()
	userID := validUserID()
	otherUserID := uuid.MustParse("00000000-0000-0000-0000-000000000002")

	// active link for target user
	link1 := makeLink(userID, "aaa111")
	// inactive link for target user — should not appear
	link2 := makeLink(userID, "bbb222")
	link2.IsActive = false
	// active link for other user — should not appear
	link3 := makeLink(otherUserID, "ccc333")

	db.Create(link1)
	db.Create(link2)
	db.Create(link3)

	results, err := repo.FindAllByUserID(ctx, userID)

	require.NoError(t, err)
	assert.Len(t, *results, 1)
	assert.Equal(t, "aaa111", (*results)[0].Code)
}

func TestRepo_FindAllByUserID_EmptyForUnknownUser(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)
	unknownID := uuid.MustParse("00000000-0000-0000-0000-000000000099")

	results, err := repo.FindAllByUserID(context.Background(), unknownID)

	require.NoError(t, err)
	assert.Empty(t, *results)
}

// --- FindByCodeAndUserID ---

func TestRepo_FindByCodeAndUserID_Found(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()
	userID := validUserID()
	link := makeLink(userID, "find123")
	db.Create(link)

	result, err := repo.FindByCodeAndUserID(ctx, "find123", userID)

	require.NoError(t, err)
	assert.Equal(t, "find123", result.Code)
	assert.Equal(t, userID, result.UserID)
}

func TestRepo_FindByCodeAndUserID_WrongUser(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()
	ownerID := validUserID()
	intruderID := uuid.MustParse("00000000-0000-0000-0000-000000000002")

	link := makeLink(ownerID, "secret")
	db.Create(link)

	result, err := repo.FindByCodeAndUserID(ctx, "secret", intruderID)

	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestRepo_FindByCodeAndUserID_InactiveNotReturned(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()
	userID := validUserID()

	link := makeLink(userID, "inactive")
	link.IsActive = false
	db.Create(link)

	result, err := repo.FindByCodeAndUserID(ctx, "inactive", userID)

	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestRepo_FindByCodeAndUserID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()
	userID := validUserID()

	result, err := repo.FindByCodeAndUserID(ctx, "doesnotexist", userID)

	assert.Error(t, err)
	assert.Nil(t, result)
}

// --- Create ---

func TestRepo_Create_Success(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()
	userID := validUserID()

	link := &Link{
		UserID:      userID,
		Code:        "newcode",
		OriginalURL: "https://example.com",
		IsActive:    true,
	}

	err := repo.Create(ctx, link)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, link.ID) // BeforeCreate sets ID

	var found Link
	db.First(&found, "code = ?", "newcode")
	assert.Equal(t, "https://example.com", found.OriginalURL)
}

func TestRepo_Create_DuplicateCodeFails(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()
	userID := validUserID()

	link1 := makeLink(userID, "dupcode")
	db.Create(link1)

	link2 := &Link{
		UserID:      userID,
		Code:        "dupcode", // same unique code — must fail on Postgres uniqueIndex
		OriginalURL: "https://other.com",
		IsActive:    true,
	}
	err := repo.Create(ctx, link2)

	assert.Error(t, err)
	// Postgres surfaces this as error code 23505; assert the service-level
	// collision detection in service_test.go covers the label increment path.
}

func TestRepo_Create_SetsCreatedAt(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()
	userID := validUserID()

	before := time.Now().Add(-time.Second)
	link := &Link{
		UserID:      userID,
		Code:        "timecode",
		OriginalURL: "https://example.com",
		IsActive:    true,
	}
	err := repo.Create(ctx, link)
	require.NoError(t, err)

	assert.True(t, link.CreatedAt.After(before))
}

// --- Update ---

func TestRepo_Update_Success(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()
	userID := validUserID()

	link := makeLink(userID, "updateme")
	db.Create(link)

	exp := time.Now().Add(24 * time.Hour).UTC().Truncate(time.Microsecond) // Postgres stores µs precision
	link.ExpiresAt = &exp
	link.OriginalURL = "https://updated.com"

	err := repo.Update(ctx, link)
	require.NoError(t, err)

	var found Link
	db.First(&found, "code = ?", "updateme")
	assert.Equal(t, "https://updated.com", found.OriginalURL)
	assert.NotNil(t, found.ExpiresAt)
}

func TestRepo_Update_SoftDelete(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()
	userID := validUserID()

	link := makeLink(userID, "deleteme")
	db.Create(link)

	link.IsActive = false
	err := repo.Update(ctx, link)
	require.NoError(t, err)

	var found Link
	db.First(&found, "code = ?", "deleteme")
	assert.False(t, found.IsActive)
}

func TestRepo_Update_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)
	ctx := context.Background()

	ghost := &Link{
		ID:          uuid.MustParse("00000000-0000-0000-0000-000000000099"),
		Code:        "ghost",
		OriginalURL: "https://ghost.com",
		IsActive:    false,
	}

	err := repo.Update(ctx, ghost)
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

// --- Integration: Full service + repo flow ---

func TestIntegration_CreateAndGetAll(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)
	svc := NewService(repo)
	ctx := context.Background()
	userID := validUserID()

	_, err := svc.Create(ctx, userID, CreateRequest{OriginalURL: "https://example.com"})
	require.NoError(t, err)

	_, err = svc.Create(ctx, userID, CreateRequest{OriginalURL: "https://another.com", CustomAlias: "custom1"})
	require.NoError(t, err)

	links, err := svc.GetAll(ctx, userID)
	require.NoError(t, err)
	assert.Len(t, links, 2)
}

func TestIntegration_CreateAndDelete(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)
	svc := NewService(repo)
	ctx := context.Background()
	userID := validUserID()

	dto, err := svc.Create(ctx, userID, CreateRequest{OriginalURL: "https://example.com", CustomAlias: "todelete"})
	require.NoError(t, err)

	err = svc.Delete(ctx, userID, dto.Code)
	require.NoError(t, err)

	links, err := svc.GetAll(ctx, userID)
	require.NoError(t, err)
	assert.Empty(t, links) // soft-deleted — filtered by is_active = true
}

func TestIntegration_CreateAndUpdateExpiry(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)
	svc := NewService(repo)
	ctx := context.Background()
	userID := validUserID()

	dto, err := svc.Create(ctx, userID, CreateRequest{OriginalURL: "https://example.com", CustomAlias: "explink"})
	require.NoError(t, err)
	assert.Nil(t, dto.ExpiresAt)

	exp := time.Now().Add(72 * time.Hour)
	err = svc.UpdateExpiry(ctx, userID, dto.Code, UpdateExpiryDto{ExpiresAt: &exp})
	require.NoError(t, err)

	// Verify persisted in Postgres
	found, err := repo.FindByCodeAndUserID(ctx, dto.Code, userID)
	require.NoError(t, err)
	assert.NotNil(t, found.ExpiresAt)
}

func TestIntegration_DeleteByWrongUser(t *testing.T) {
	db := setupTestDB(t)
	repo := NewRepository(db)
	svc := NewService(repo)
	ctx := context.Background()
	ownerID := validUserID()
	attackerID := uuid.MustParse("00000000-0000-0000-0000-000000000002")

	dto, err := svc.Create(ctx, ownerID, CreateRequest{OriginalURL: "https://example.com", CustomAlias: "owned"})
	require.NoError(t, err)

	err = svc.Delete(ctx, attackerID, dto.Code)
	assert.EqualError(t, err, "link not found or unauthorized")

	// Verify the link is still active and visible to its owner
	links, _ := svc.GetAll(ctx, ownerID)
	assert.Len(t, links, 1)
}