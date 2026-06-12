package link

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) FindAllByUserID(ctx context.Context, userID uuid.UUID, pagination Pagination) (*[]Link, error) {
	args := m.Called(ctx, userID, pagination)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*[]Link), args.Error(1)
}

func (m *MockRepository) FindByCodeAndUserID(ctx context.Context, code string, userID uuid.UUID) (*Link, error) {
	args := m.Called(ctx, code, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Link), args.Error(1)
}

func (m *MockRepository) Create(ctx context.Context, link *Link) error {
	args := m.Called(ctx, link)
	return args.Error(0)
}

func (m *MockRepository) Update(ctx context.Context, link *Link) error {
	args := m.Called(ctx, link)
	return args.Error(0)
}

// --- GetAll Tests ---

func TestService_GetAll_Success(t *testing.T) {
	repo := new(MockRepository)
	svc := NewService(repo)
	userID := validUserID()
	now := time.Now()

	links := &[]Link{
		{Code: "abc", OriginalURL: "https://example.com", IsActive: true, CreatedAt: now},
		{Code: "def", OriginalURL: "https://another.com", IsActive: true, CreatedAt: now},
	}
	repo.On("FindAllByUserID", mock.Anything, userID, mock.Anything).Return(links, nil)

	dtos, err := svc.GetAll(context.Background(), userID)

	assert.NoError(t, err)
	assert.Len(t, dtos, 2)
	assert.Equal(t, "abc", dtos[0].Code)
	assert.Equal(t, "def", dtos[1].Code)
	repo.AssertExpectations(t)
}

func TestService_GetAll_Empty(t *testing.T) {
	repo := new(MockRepository)
	svc := NewService(repo)
	userID := validUserID()

	repo.On("FindAllByUserID", mock.Anything, userID, mock.Anything).Return(&[]Link{}, nil)

	dtos, err := svc.GetAll(context.Background(), userID)

	assert.NoError(t, err)
	assert.Nil(t, dtos) // nil slice when no links exist
	repo.AssertExpectations(t)
}

func TestService_GetAll_RepoError(t *testing.T) {
	repo := new(MockRepository)
	svc := NewService(repo)
	userID := validUserID()

	repo.On("FindAllByUserID", mock.Anything, userID, mock.Anything).Return(nil, errors.New("db connection failed"))

	dtos, err := svc.GetAll(context.Background(), userID)

	assert.Error(t, err)
	assert.Nil(t, dtos)
	repo.AssertExpectations(t)
}

// --- Create Tests ---

func TestService_Create_WithRandomCode(t *testing.T) {
	repo := new(MockRepository)
	svc := NewService(repo)
	userID := validUserID()

	repo.On("Create", mock.Anything, mock.AnythingOfType("*link.Link")).Return(nil)

	req := CreateRequest{OriginalURL: "https://example.com"}
	dto, err := svc.Create(context.Background(), userID, req)

	assert.NoError(t, err)
	assert.NotNil(t, dto)
	assert.Len(t, dto.Code, 8) // random code length
	assert.Equal(t, "https://example.com", dto.OriginalURL)
	assert.True(t, dto.IsActive)
	repo.AssertExpectations(t)
}

func TestService_Create_WithCustomAlias(t *testing.T) {
	repo := new(MockRepository)
	svc := NewService(repo)
	userID := validUserID()

	repo.On("Create", mock.Anything, mock.AnythingOfType("*link.Link")).Return(nil)

	req := CreateRequest{OriginalURL: "https://example.com", CustomAlias: "mylink"}
	dto, err := svc.Create(context.Background(), userID, req)

	assert.NoError(t, err)
	assert.NotNil(t, dto)
	assert.Equal(t, "mylink", dto.Code)
	assert.NotNil(t, dto.CustomAlias)
	assert.Equal(t, "mylink", *dto.CustomAlias)
	repo.AssertExpectations(t)
}

func TestService_Create_WithExpiry(t *testing.T) {
	repo := new(MockRepository)
	svc := NewService(repo)
	userID := validUserID()
	exp := time.Now().Add(48 * time.Hour)

	repo.On("Create", mock.Anything, mock.AnythingOfType("*link.Link")).Return(nil)

	req := CreateRequest{OriginalURL: "https://example.com", ExpiresAt: &exp}
	dto, err := svc.Create(context.Background(), userID, req)

	assert.NoError(t, err)
	assert.NotNil(t, dto.ExpiresAt)
	repo.AssertExpectations(t)
}

func TestService_Create_RepoError(t *testing.T) {
	repo := new(MockRepository)
	svc := NewService(repo)
	userID := validUserID()

	repo.On("Create", mock.Anything, mock.AnythingOfType("*link.Link")).Return(errors.New("duplicate key value violates unique constraint"))

	req := CreateRequest{OriginalURL: "https://example.com"}
	dto, err := svc.Create(context.Background(), userID, req)

	assert.Error(t, err)
	assert.Nil(t, dto)
	repo.AssertExpectations(t)
}

func TestService_Create_DuplicateCustomAlias(t *testing.T) {
	repo := new(MockRepository)
	svc := NewService(repo)
	userID := validUserID()

	repo.On("Create", mock.Anything, mock.AnythingOfType("*link.Link")).Return(errors.New("duplicate key 23505"))

	req := CreateRequest{OriginalURL: "https://example.com", CustomAlias: "taken"}
	dto, err := svc.Create(context.Background(), userID, req)

	assert.Error(t, err)
	assert.Nil(t, dto)
	repo.AssertExpectations(t)
}

// --- UpdateExpiry Tests ---

func TestService_UpdateExpiry_Success(t *testing.T) {
	repo := new(MockRepository)
	svc := NewService(repo)
	userID := validUserID()
	code := "abc123"
	now := time.Now()
	exp := now.Add(24 * time.Hour)

	existing := &Link{Code: code, UserID: userID, IsActive: true}
	repo.On("FindByCodeAndUserID", mock.Anything, code, userID).Return(existing, nil)
	repo.On("Update", mock.Anything, mock.MatchedBy(func(l *Link) bool {
		return l.ExpiresAt != nil && l.ExpiresAt.Equal(exp)
	})).Return(nil)

	err := svc.UpdateExpiry(context.Background(), userID, code, UpdateExpiryDto{ExpiresAt: &exp})

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestService_UpdateExpiry_NotFound(t *testing.T) {
	repo := new(MockRepository)
	svc := NewService(repo)
	userID := validUserID()
	code := "nosuchcode"
	exp := time.Now().Add(24 * time.Hour)

	repo.On("FindByCodeAndUserID", mock.Anything, code, userID).Return(nil, errors.New("record not found"))

	err := svc.UpdateExpiry(context.Background(), userID, code, UpdateExpiryDto{ExpiresAt: &exp})

	assert.Error(t, err)
	assert.EqualError(t, err, "link not found or unauthorized")
	repo.AssertNotCalled(t, "Update")
	repo.AssertExpectations(t)
}

func TestService_UpdateExpiry_UpdateFails(t *testing.T) {
	repo := new(MockRepository)
	svc := NewService(repo)
	userID := validUserID()
	code := "abc123"
	exp := time.Now().Add(24 * time.Hour)

	existing := &Link{Code: code, UserID: userID, IsActive: true}
	repo.On("FindByCodeAndUserID", mock.Anything, code, userID).Return(existing, nil)
	repo.On("Update", mock.Anything, mock.Anything).Return(errors.New("db error"))

	err := svc.UpdateExpiry(context.Background(), userID, code, UpdateExpiryDto{ExpiresAt: &exp})

	assert.Error(t, err)
	repo.AssertExpectations(t)
}

// --- Delete Tests ---

func TestService_Delete_Success(t *testing.T) {
	repo := new(MockRepository)
	svc := NewService(repo)
	userID := validUserID()
	code := "abc123"

	existing := &Link{Code: code, UserID: userID, IsActive: true}
	repo.On("FindByCodeAndUserID", mock.Anything, code, userID).Return(existing, nil)
	repo.On("Update", mock.Anything, mock.MatchedBy(func(l *Link) bool {
		return !l.IsActive // soft delete
	})).Return(nil)

	err := svc.Delete(context.Background(), userID, code)

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestService_Delete_NotFound(t *testing.T) {
	repo := new(MockRepository)
	svc := NewService(repo)
	userID := validUserID()
	code := "nosuchcode"

	repo.On("FindByCodeAndUserID", mock.Anything, code, userID).Return(nil, errors.New("record not found"))

	err := svc.Delete(context.Background(), userID, code)

	assert.Error(t, err)
	assert.EqualError(t, err, "link not found or unauthorized")
	repo.AssertNotCalled(t, "Update")
	repo.AssertExpectations(t)
}

func TestService_Delete_UpdateFails(t *testing.T) {
	repo := new(MockRepository)
	svc := NewService(repo)
	userID := validUserID()
	code := "abc123"

	existing := &Link{Code: code, UserID: userID, IsActive: true}
	repo.On("FindByCodeAndUserID", mock.Anything, code, userID).Return(existing, nil)
	repo.On("Update", mock.Anything, mock.Anything).Return(errors.New("db error"))

	err := svc.Delete(context.Background(), userID, code)

	assert.Error(t, err)
	repo.AssertExpectations(t)
}

// --- generateRandomCode indirectly tested via Create ---

func TestService_Create_RandomCodeIsAlphanumeric(t *testing.T) {
	repo := new(MockRepository)
	svc := NewService(repo)
	userID := validUserID()

	var capturedLink *Link
	repo.On("Create", mock.Anything, mock.AnythingOfType("*link.Link")).
		Run(func(args mock.Arguments) {
			capturedLink = args.Get(1).(*Link)
		}).
		Return(nil)

	req := CreateRequest{OriginalURL: "https://example.com"}
	_, err := svc.Create(context.Background(), userID, req)

	assert.NoError(t, err)
	assert.Len(t, capturedLink.Code, 8)
	for _, ch := range capturedLink.Code {
		assert.True(t,
			(ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9'),
			"code contains non-alphanumeric character: %c", ch,
		)
	}
}

func TestService_Create_UniqueCodesGenerated(t *testing.T) {
	// Generate multiple codes and assert no immediate collisions (probabilistic)
	repo := new(MockRepository)
	svc := NewService(repo)
	userID := validUserID()
	repo.On("Create", mock.Anything, mock.Anything).Return(nil)

	codes := make(map[string]bool)
	for i := 0; i < 50; i++ {
		dto, err := svc.Create(context.Background(), userID, CreateRequest{OriginalURL: "https://example.com"})
		assert.NoError(t, err)
		codes[dto.Code] = true
	}

	// With 62^8 possibilities, 50 samples should be unique
	assert.Equal(t, 50, len(codes))
}