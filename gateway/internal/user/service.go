package user

import (
	"context"
	"errors"

	"aziz.dev/gateway/internal/security"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
)

type Service interface {
	Register(ctx context.Context, r *RegisterRequest) (error)
	Login(ctx context.Context, r *LoginRequest) (*security.TokenPair, error)
	Refresh(ctx context.Context) (*security.TokenPair, error)
}

type service struct {
	repo Repository
	redisClient *redis.Client
	jwtService security.Service
}

func NewService(repo Repository, redisClient *redis.Client, jwtService security.Service) Service {
	return &service{
		repo: repo,
		redisClient: redisClient,
		jwtService: jwtService,
	}
}

func (s *service) Register(ctx context.Context, r *RegisterRequest) error {
	existing, err := s.repo.FindByEmail(ctx, r.Email)
	if err != nil {
		return err
	}
	if existing != nil {
		return errors.New("email already in use")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(r.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	user := User{
		Email: r.Email,
		PasswordHash: string(hashedPassword),
		IsActive: true,
	}

	err = s.repo.Create(ctx, user)
	if err != nil {
		return err
	}
	
	return nil
}

//TODO: return access token + refresh token, store in redis
func (s *service) Login(ctx context.Context, r *LoginRequest) (*security.TokenPair, error) {
	user, err := s.repo.FindByEmail(ctx, r.Email)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errors.New("user not found")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(r.Password)); err != nil {
		return nil, errors.New("invalid password")
	}

	if !user.IsActive {
		return nil, errors.New("account is not active")
	}

	accessToken, err := s.jwtService.GenerateAccessToken(user.ID.String())
	if err != nil {
		return nil, err
	}

	refreshToken, err := s.jwtService.GenerateRefreshToken(user.ID.String())
	if err != nil {
		return nil, err
	}
	
	return &security.TokenPair{
		AccessToken: accessToken,
		RefreshToken: refreshToken,
	}, nil
}

//TODO: validate refresh token, generate new access token + refresh token, store in redis
func (s *service) Refresh(ctx context.Context) (*security.TokenPair, error) {
	return nil, nil
}