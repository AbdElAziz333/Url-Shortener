package user

import (
	"context"
	"errors"

	"aziz.dev/gateway/internal/security"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
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
	log := logrus.WithField("email", r.Email)
	log.Info("Registering new user")

	existing, err := s.repo.FindByEmail(ctx, r.Email)
	if err != nil {
		log.WithError(err).Error("Failed to check existing user")
		return err
	}
	if existing != nil {
		log.Warn("Email already in use")
		return errors.New("email already in use")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(r.Password), bcrypt.DefaultCost)
	if err != nil {
		log.WithError(err).Error("Failed to hash password")
		return err
	}

	user := User{
		Email: r.Email,
		PasswordHash: string(hashedPassword),
		IsActive: true,
	}

	err = s.repo.Create(ctx, user)
	if err != nil {
		log.WithError(err).Error("Failed to create user")
		return err
	}

	log.Info("User registered successfully")
	return nil
}

//TODO: return access token + refresh token, store in redis
func (s *service) Login(ctx context.Context, r *LoginRequest) (*security.TokenPair, error) {
	log := logrus.WithField("email", r.Email)
	log.Info("Attempting user login")

	user, err := s.repo.FindByEmail(ctx, r.Email)
	if err != nil {
		log.WithError(err).Error("Failed to find user by email")
		return nil, err
	}
	if user == nil {
		log.Warn("User not found")
		return nil, errors.New("user not found")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(r.Password)); err != nil {
		log.Warn("Invalid password attempt")
		return nil, errors.New("invalid password")
	}

	if !user.IsActive {
		log.Warn("Account not active")
		return nil, errors.New("account is not active")
	}

	accessToken, err := s.jwtService.GenerateAccessToken(user.ID.String())
	if err != nil {
		log.WithError(err).Error("Failed to generate access token")
		return nil, err
	}

	refreshToken, err := s.jwtService.GenerateRefreshToken(user.ID.String())
	if err != nil {
		log.WithError(err).Error("Failed to generate refresh token")
		return nil, err
	}

	log.Info("User logged in successfully")
	return &security.TokenPair{
		AccessToken: accessToken,
		RefreshToken: refreshToken,
	}, nil
}

//TODO: validate refresh token, generate new access token + refresh token, store in redis
func (s *service) Refresh(ctx context.Context) (*security.TokenPair, error) {
	logrus.Info("Refreshing tokens")
	return nil, nil
}