package link

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestLink_BeforeCreate_SetsUUID(t *testing.T) {
	db := setupTestDB(t)

	userID := validUserID()
	link := &Link{
		UserID:      userID,
		Code:        "hooktest1",
		OriginalURL: "https://example.com",
		IsActive:    true,
	}

	assert.Equal(t, uuid.Nil, link.ID, "ID should be nil before create")
	db.Create(link)
	assert.NotEqual(t, uuid.Nil, link.ID, "ID should be set after create")
}

func TestLink_BeforeCreate_SetsCreatedAt(t *testing.T) {
	db := setupTestDB(t)

	before := time.Now().Add(-time.Second)
	link := &Link{
		UserID:      validUserID(),
		Code:        "hooktest2",
		OriginalURL: "https://example.com",
		IsActive:    true,
	}

	db.Create(link)
	assert.True(t, link.CreatedAt.After(before), "CreatedAt should be set to approximately now")
}

func TestLink_BeforeCreate_IDIsUUIDv7(t *testing.T) {
	db := setupTestDB(t)

	link := &Link{
		UserID:      validUserID(),
		Code:        "hooktest3",
		OriginalURL: "https://example.com",
		IsActive:    true,
	}
	db.Create(link)

	// UUID v7 has version bits: first nibble of 3rd group should be 7
	assert.Equal(t, byte(7), link.ID[6]>>4, "ID should be a UUIDv7")
}

func TestLink_BeforeCreate_UniqueIDsPerRecord(t *testing.T) {
	db := setupTestDB(t)

	link1 := &Link{UserID: validUserID(), Code: "uid1", OriginalURL: "https://a.com", IsActive: true}
	link2 := &Link{UserID: validUserID(), Code: "uid2", OriginalURL: "https://b.com", IsActive: true}

	db.Create(link1)
	db.Create(link2)

	assert.NotEqual(t, link1.ID, link2.ID)
}

// --- mapToDto ---

func TestMapToDto_AllFieldsMapped(t *testing.T) {
	exp := time.Now().Add(24 * time.Hour)
	now := time.Now()
	link := &Link{
		Code:        "abc123",
		OriginalURL: "https://example.com",
		CustomAlias: "myalias",
		ExpiresAt:   &exp,
		IsActive:    true,
		CreatedAt:   now,
	}

	dto := mapToDto(link)

	assert.Equal(t, "abc123", dto.Code)
	assert.Equal(t, "https://example.com", dto.OriginalURL)
	assert.Equal(t, "myalias", dto.CustomAlias)
	assert.Equal(t, &exp, dto.ExpiresAt)
	assert.True(t, dto.IsActive)
	assert.Equal(t, now, dto.CreatedAt)
}

func TestMapToDto_NilExpiry(t *testing.T) {
	link := &Link{
		Code:        "noexpiry",
		OriginalURL: "https://example.com",
		ExpiresAt:   nil,
		IsActive:    true,
		CreatedAt:   time.Now(),
	}

	dto := mapToDto(link)

	assert.Nil(t, dto.ExpiresAt)
}

func TestMapToDto_InactiveLink(t *testing.T) {
	link := &Link{
		Code:        "inactive",
		OriginalURL: "https://example.com",
		IsActive:    false,
		CreatedAt:   time.Now(),
	}

	dto := mapToDto(link)

	assert.False(t, dto.IsActive)
}