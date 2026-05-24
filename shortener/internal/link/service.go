package link

import (
	"context"
	"crypto/rand"
	"errors"
	"math/big"
	"strings"
	"time"

	"aziz.dev/shortener/internal/middleware"
	"github.com/google/uuid"
)

type Service interface {
	GetAll(ctx context.Context, userID uuid.UUID) ([]Dto, error)
	Create(ctx context.Context, userID uuid.UUID, req CreateRequest) (*Dto, error)
	UpdateExpiry(ctx context.Context, userID uuid.UUID, code string, req UpdateExpiryDto) error
	// UpdateCustomAlias(ctx context.Context, userID uuid.UUID, code string, req UpdateAliasDto) error
	Delete(ctx context.Context, userID uuid.UUID, code string) error
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{
		repo: repo,
	}
}

func (s *service) GetAll(ctx context.Context, userID uuid.UUID) ([]Dto, error) {
	links, err := s.repo.FindAllByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	var dtos []Dto
	for _, l := range *links {
		dtos = append(dtos, mapToDto(&l))
	}

	return dtos, nil
}

func (s *service) Create(ctx context.Context, userID uuid.UUID, req CreateRequest) (*Dto, error) {
	code := req.CustomAlias
	if code == "" {
		var err error
		code, err = generateRandomCode(8)

		if err != nil {
			middleware.UrlsCreatedTotal.WithLabelValues("failure").Inc()
			return nil, err
		}
	}

	link := &Link{
		UserID:      userID,
		Code:        code,
		OriginalURL: req.OriginalURL,
		CustomAlias: req.CustomAlias,
		ExpiresAt:   req.ExpiresAt,
		IsActive:    true,
		CreatedAt:   time.Now(),
	}

	if err := s.repo.Create(ctx, link); err != nil {
		middleware.UrlsCreatedTotal.WithLabelValues("failure").Inc()
		if req.CustomAlias == "" && (strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "23505")) {
			middleware.HashCollisionsTotal.Inc()
		}
		return nil, err
	}

	middleware.UrlsCreatedTotal.WithLabelValues("success").Inc()

	return &Dto{
		Code:        link.Code,
		OriginalURL: link.OriginalURL,
		CustomAlias: link.CustomAlias,
		ExpiresAt:   link.ExpiresAt,
		IsActive:    link.IsActive,
		CreatedAt:   link.CreatedAt,
	}, nil
}

func (s *service) UpdateExpiry(ctx context.Context, userID uuid.UUID, code string, req UpdateExpiryDto) error {
	link, err := s.repo.FindByCodeAndUserID(ctx, code, userID)
	if err != nil {
		return errors.New("link not found or unauthorized")
	}

	link.ExpiresAt = req.ExpiresAt
	return s.repo.Update(ctx, link)
}

// func (s *service) UpdateCustomAlias(ctx context.Context, userID uuid.UUID, code string, req UpdateAliasDto) error {
// 	link, err := s.repo.FindByCodeAndUserID(ctx, code, userID)
// 	if err != nil {
// 		return errors.New("link not found or unauthorized")
// 	}

// 	link.CustomAlias = req.CustomAlias
// 	link.Code = req.CustomAlias // Changing the code as well since alias acts as code
// 	return s.repo.UpdateWithOutbox(ctx, link, nil)
// }

func (s *service) Delete(ctx context.Context, userID uuid.UUID, code string) error {
	link, err := s.repo.FindByCodeAndUserID(ctx, code, userID)
	if err != nil {
		return errors.New("link not found or unauthorized")
	}

	link.IsActive = false

	return s.repo.Update(ctx, link)
}

func generateRandomCode(length int) (string, error) {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", err
		}
		b[i] = charset[n.Int64()]
	}
	return string(b), nil
}