package user

import (
	"context"
	"errors"

	"aziz.dev/gateway/jwt"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
)

type Service interface {
	Register(ctx context.Context, r *RegisterRequest) (error)
	Login(ctx context.Context, r *LoginRequest) (error)
	Refresh(ctx context.Context) (string, error)
}

type service struct {
	repo Repository
	redisClient *redis.Client
	jwtService jwt.Service
}

func NewService(repo Repository, redisClient *redis.Client, jwtService jwt.Service) Service {
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

	user := &User{
		FullName: r.FullName,
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
func (s *service) Login(ctx context.Context, r *LoginRequest) error {
	user, err := s.repo.FindByEmail(ctx, r.Email)
	if err != nil {
		return err
	}
	if user == nil {
		return errors.New("user not found")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(r.Password)); err != nil {
		return errors.New("invalid password")
	}

	if !user.IsActive {
		return errors.New("account is not active")
	}
	
	return nil
}

//TODO: validate refresh token, generate new access token + refresh token, store in redis
func (s *service) Refresh(ctx context.Context) (string, error) {
	return "", nil
}