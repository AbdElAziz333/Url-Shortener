package link

import (
	"context"
	"crypto/rand"
	"errors"
	"math/big"
	"strings"
	"time"

	sqlcgen "aziz.dev/shortener/internal/postgres/sqlc"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
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
	logrus.WithField("user_id", userID).Info("Fetching all links for user")
	links, err := s.repo.FindAllByUserID(ctx, userID, Pagination{Page: 1, PageSize: 10})
	if err != nil {
		logrus.WithError(err).WithField("user_id", userID).Error("Failed to fetch links")
		return nil, err
	}

	var dtos []Dto
	for _, l := range links {
		dtos = append(dtos, mapToDto(l))
	}

	logrus.WithField("user_id", userID).WithField("count", len(dtos)).Info("Successfully fetched links")
	return dtos, nil
}

func (s *service) Create(ctx context.Context, userID uuid.UUID, req CreateRequest) (*Dto, error) {
	logrus.WithField("user_id", userID).WithField("original_url", req.OriginalURL).Info("Creating new link")
	code := req.CustomAlias
	if code == "" {
		var err error
		code, err = generateRandomCode(8)

		if err != nil {
			logrus.WithError(err).Error("Failed to generate random code")
			return nil, err
		}
		
		logrus.WithField("code", code).Debug("Generated random code")
	}

	// Only store the alias when one was actually provided.
	// Storing "" would violate the UNIQUE constraint since Postgres treats
	// all empty strings as equal, while multiple NULLs are allowed.
	var customAlias *string
	if req.CustomAlias != "" {
		customAlias = &req.CustomAlias
	}

	arg := sqlcgen.CreateLinkParams{
		UserID:      userID,
		Code:        code,
		OriginalUrl: req.OriginalURL,
		CustomAlias: customAlias,
		ExpiresAt:   req.ExpiresAt,
		IsActive:    true,
		CreatedAt:   time.Now(),
	}

	result, err := s.repo.Create(ctx, arg)

	if err != nil {
		if req.CustomAlias == "" && (strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "23505")) {
			logrus.WithField("code", code).Warn("Hash collision detected")
		}

		logrus.WithError(err).Error("Failed to create link")
		return nil, err
	}

	logrus.WithField("user_id", userID).WithField("code", code).Info("Successfully created link")

	return &Dto{
		Code:        result.Code,
		OriginalURL: result.OriginalUrl,
		CustomAlias: result.CustomAlias,
		ExpiresAt:   result.ExpiresAt,
		IsActive:    result.IsActive,
		CreatedAt:   result.CreatedAt,
	}, nil
}

func (s *service) UpdateExpiry(ctx context.Context, userID uuid.UUID, code string, req UpdateExpiryDto) error {
	logrus.WithField("user_id", userID).WithField("code", code).Info("Updating link expiry")
	link, err := s.repo.FindByCodeAndUserID(ctx, code, userID)
	if err != nil {
		logrus.WithError(err).WithField("user_id", userID).WithField("code", code).Error("Failed to find link")
		return errors.New("link not found or unauthorized")
	}

	link.ExpiresAt = req.ExpiresAt
	arg := sqlcgen.UpdateLinkParams{
		ExpiresAt: link.ExpiresAt,
	}

	id, err := s.repo.Update(ctx, arg)

	if err != nil {
		logrus.WithError(err).Error("Failed to update link with id:", id)
		return err
	}

	logrus.WithField("user_id", userID).WithField("code", code).Info("Successfully updated link expiry")
	return nil
}

func (s *service) Delete(ctx context.Context, userID uuid.UUID, code string) error {
    logrus.WithField("user_id", userID).WithField("code", code).Info("Deleting link")
    
    link, err := s.repo.FindByCodeAndUserID(ctx, code, userID)
    if err != nil {
        logrus.WithError(err).WithField("user_id", userID).WithField("code", code).Error("Failed to find link")
        return errors.New("link not found or unauthorized")
    }

    now := time.Now().UTC()

    arg := sqlcgen.UpdateLinkParams{
        ID:          link.ID,
        Code:        link.Code,
        OriginalUrl: link.OriginalUrl,
        CustomAlias: link.CustomAlias,
        ExpiresAt:   &now,
        IsActive:    false, // Soft-delete
    }

    rowsAffected, err := s.repo.Update(ctx, arg)
    if err != nil {
        logrus.WithError(err).Error("Failed to delete link")
        return err
    }

    if rowsAffected == 0 {
        return errors.New("link not found or already deleted")
    }

    logrus.WithField("user_id", userID).WithField("code", code).Info("Successfully deleted link")
    return nil
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
