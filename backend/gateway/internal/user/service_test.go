package user

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/crypto/bcrypt"
)

// --- Mock Repository ---

type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) FindByEmail(ctx context.Context, email string) (*User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*User), args.Error(1)
}

func (m *MockRepository) Create(ctx context.Context, u User) error {
	args := m.Called(ctx, u)
	return args.Error(0)
}

// --- Mock JWT Service ---

type MockJWTService struct {
	mock.Mock
}

func (m *MockJWTService) GenerateAccessToken(userID string) (string, error) {
	args := m.Called(userID)
	return args.String(0), args.Error(1)
}

func (m *MockJWTService) GenerateRefreshToken(userID string) (string, error) {
	args := m.Called(userID)
	return args.String(0), args.Error(1)
}

// --- Register Tests ---

func TestService_Register_Success(t *testing.T) {
	repo := new(MockRepository)
	jwt := new(MockJWTService)
	repo.On("FindByEmail", mock.Anything, "new@example.com").Return(nil, nil)
	repo.On("Create", mock.Anything, mock.AnythingOfType("User")).Return(nil)

	svc := NewService(repo, nil, jwt)
	err := svc.Register(context.Background(), &RegisterRequest{
		Email:    "new@example.com",
		Password: "securePass1!",
	})

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestService_Register_EmailAlreadyInUse(t *testing.T) {
	repo := new(MockRepository)
	jwt := new(MockJWTService)
	existing := &User{Email: "taken@example.com"}
	repo.On("FindByEmail", mock.Anything, "taken@example.com").Return(existing, nil)

	svc := NewService(repo, nil, jwt)
	err := svc.Register(context.Background(), &RegisterRequest{
		Email:    "taken@example.com",
		Password: "pass",
	})

	assert.EqualError(t, err, "email already in use")
	repo.AssertNotCalled(t, "Create")
}

func TestService_Register_FindByEmailError(t *testing.T) {
	repo := new(MockRepository)
	jwt := new(MockJWTService)
	repo.On("FindByEmail", mock.Anything, "err@example.com").Return(nil, errors.New("db error"))

	svc := NewService(repo, nil, jwt)
	err := svc.Register(context.Background(), &RegisterRequest{
		Email:    "err@example.com",
		Password: "pass",
	})

	assert.EqualError(t, err, "db error")
	repo.AssertNotCalled(t, "Create")
}

func TestService_Register_CreateError(t *testing.T) {
	repo := new(MockRepository)
	jwt := new(MockJWTService)
	repo.On("FindByEmail", mock.Anything, "new@example.com").Return(nil, nil)
	repo.On("Create", mock.Anything, mock.AnythingOfType("User")).Return(errors.New("insert failed"))

	svc := NewService(repo, nil, jwt)
	err := svc.Register(context.Background(), &RegisterRequest{
		Email:    "new@example.com",
		Password: "pass",
	})

	assert.EqualError(t, err, "insert failed")
}

func TestService_Register_PasswordIsHashed(t *testing.T) {
	repo := new(MockRepository)
	jwt := new(MockJWTService)
	repo.On("FindByEmail", mock.Anything, mock.Anything).Return(nil, nil)

	var capturedUser User
	repo.On("Create", mock.Anything, mock.AnythingOfType("User")).
		Run(func(args mock.Arguments) {
			capturedUser = args.Get(1).(User)
		}).
		Return(nil)

	svc := NewService(repo, nil, jwt)
	_ = svc.Register(context.Background(), &RegisterRequest{
		Email:    "hash@example.com",
		Password: "plaintext",
	})

	assert.NotEqual(t, "plaintext", capturedUser.PasswordHash)
	err := bcrypt.CompareHashAndPassword([]byte(capturedUser.PasswordHash), []byte("plaintext"))
	assert.NoError(t, err, "stored hash should match original password")
}

func TestService_Register_UserIsActive(t *testing.T) {
	repo := new(MockRepository)
	jwt := new(MockJWTService)
	repo.On("FindByEmail", mock.Anything, mock.Anything).Return(nil, nil)

	var capturedUser User
	repo.On("Create", mock.Anything, mock.AnythingOfType("User")).
		Run(func(args mock.Arguments) {
			capturedUser = args.Get(1).(User)
		}).
		Return(nil)

	svc := NewService(repo, nil, jwt)
	_ = svc.Register(context.Background(), &RegisterRequest{
		Email:    "active@example.com",
		Password: "pass",
	})

	assert.True(t, capturedUser.IsActive)
}

// --- Login Tests ---

func newHashedUser(email, password string) *User {
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	return &User{
		ID:           uuid.Must(uuid.NewV7()),
		Email:        email,
		PasswordHash: string(hash),
		IsActive:     true,
	}
}

func TestService_Login_Success(t *testing.T) {
	repo := new(MockRepository)
	jwtSvc := new(MockJWTService)
	u := newHashedUser("user@example.com", "correctPass")

	repo.On("FindByEmail", mock.Anything, "user@example.com").Return(u, nil)
	jwtSvc.On("GenerateAccessToken", u.ID.String()).Return("access-token", nil)
	jwtSvc.On("GenerateRefreshToken", u.ID.String()).Return("refresh-token", nil)

	svc := NewService(repo, nil, jwtSvc)
	pair, err := svc.Login(context.Background(), &LoginRequest{
		Email:    "user@example.com",
		Password: "correctPass",
	})

	assert.NoError(t, err)
	assert.Equal(t, "access-token", pair.AccessToken)
	assert.Equal(t, "refresh-token", pair.RefreshToken)
	repo.AssertExpectations(t)
	jwtSvc.AssertExpectations(t)
}

func TestService_Login_UserNotFound(t *testing.T) {
	repo := new(MockRepository)
	jwtSvc := new(MockJWTService)
	repo.On("FindByEmail", mock.Anything, "ghost@example.com").Return(nil, nil)

	svc := NewService(repo, nil, jwtSvc)
	pair, err := svc.Login(context.Background(), &LoginRequest{
		Email:    "ghost@example.com",
		Password: "pass",
	})

	assert.Nil(t, pair)
	assert.EqualError(t, err, "user not found")
}

func TestService_Login_WrongPassword(t *testing.T) {
	repo := new(MockRepository)
	jwtSvc := new(MockJWTService)
	u := newHashedUser("user@example.com", "correctPass")
	repo.On("FindByEmail", mock.Anything, "user@example.com").Return(u, nil)

	svc := NewService(repo, nil, jwtSvc)
	pair, err := svc.Login(context.Background(), &LoginRequest{
		Email:    "user@example.com",
		Password: "wrongPass",
	})

	assert.Nil(t, pair)
	assert.EqualError(t, err, "invalid password")
}

func TestService_Login_InactiveAccount(t *testing.T) {
	repo := new(MockRepository)
	jwtSvc := new(MockJWTService)
	u := newHashedUser("user@example.com", "pass")
	u.IsActive = false
	repo.On("FindByEmail", mock.Anything, "user@example.com").Return(u, nil)

	svc := NewService(repo, nil, jwtSvc)
	pair, err := svc.Login(context.Background(), &LoginRequest{
		Email:    "user@example.com",
		Password: "pass",
	})

	assert.Nil(t, pair)
	assert.EqualError(t, err, "account is not active")
}

func TestService_Login_FindByEmailError(t *testing.T) {
	repo := new(MockRepository)
	jwtSvc := new(MockJWTService)
	repo.On("FindByEmail", mock.Anything, "user@example.com").Return(nil, errors.New("db error"))

	svc := NewService(repo, nil, jwtSvc)
	pair, err := svc.Login(context.Background(), &LoginRequest{
		Email:    "user@example.com",
		Password: "pass",
	})

	assert.Nil(t, pair)
	assert.EqualError(t, err, "db error")
}

func TestService_Login_AccessTokenGenerationError(t *testing.T) {
	repo := new(MockRepository)
	jwtSvc := new(MockJWTService)
	u := newHashedUser("user@example.com", "pass")
	repo.On("FindByEmail", mock.Anything, "user@example.com").Return(u, nil)
	jwtSvc.On("GenerateAccessToken", u.ID.String()).Return("", errors.New("signing error"))

	svc := NewService(repo, nil, jwtSvc)
	pair, err := svc.Login(context.Background(), &LoginRequest{
		Email:    "user@example.com",
		Password: "pass",
	})

	assert.Nil(t, pair)
	assert.EqualError(t, err, "signing error")
}

func TestService_Login_RefreshTokenGenerationError(t *testing.T) {
	repo := new(MockRepository)
	jwtSvc := new(MockJWTService)
	u := newHashedUser("user@example.com", "pass")
	repo.On("FindByEmail", mock.Anything, "user@example.com").Return(u, nil)
	jwtSvc.On("GenerateAccessToken", u.ID.String()).Return("access-token", nil)
	jwtSvc.On("GenerateRefreshToken", u.ID.String()).Return("", errors.New("refresh error"))

	svc := NewService(repo, nil, jwtSvc)
	pair, err := svc.Login(context.Background(), &LoginRequest{
		Email:    "user@example.com",
		Password: "pass",
	})

	assert.Nil(t, pair)
	assert.EqualError(t, err, "refresh error")
}