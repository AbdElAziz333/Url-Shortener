package link

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Repository interface {
	FindAllByUserID(ctx context.Context, userID uuid.UUID, pagination Pagination) (*[]Link, error)
	FindByCodeAndUserID(ctx context.Context, code string, userID uuid.UUID) (*Link, error)
	Create(ctx context.Context, link *Link) error
	Update(ctx context.Context, link *Link) error
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{
		db: db,
	}
}

type Pagination struct {
	Page int // e.g., 1 for the first page
	PageSize int // e.g., 10 items per page
}

func (r *repository) FindAllByUserID(ctx context.Context, userID uuid.UUID, pagination Pagination) (*[]Link, error) {
	var links []Link

	if pagination.Page <= 0 {
		pagination.Page = 1
	}

	if pagination.PageSize <= 0 {
		pagination.PageSize = 10
	}

	// Calculate offset: (page - 1) * pageSize
    // Page 1: (1 - 1) * 10 = 0 (skip 0 items)
    // Page 2: (2 - 1) * 10 = 10 (skip first 10 items)
	offset := (pagination.Page - 1) * pagination.PageSize

	err := r.db.WithContext(ctx).
				Where("user_id = ? AND is_active = ?", userID, true).
				Limit(pagination.PageSize).
				Offset(offset).
				Find(&links).
				Error

	return &links, err
}

func (r *repository) FindByCodeAndUserID(ctx context.Context, code string, userID uuid.UUID) (*Link, error) {
	var link Link
	err := r.db.WithContext(ctx).Where("code = ? AND user_id = ? AND is_active = ?", code, userID, true).First(&link).Error
	if err != nil {
		return nil, err
	}

	return &link, nil
}

func (r *repository) Create(ctx context.Context, link *Link) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(link).Error; err != nil {
			return err
		}

		return nil
	})
}

func (r *repository) Update(ctx context.Context, link *Link) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Avoid gorm.Save() here: depending on model/PK state it can fall back to INSERT,
		// which breaks delete (is_active=false) and can violate PK constraints.
		updates := map[string]any{
			"code":         link.Code,
			"original_url": link.OriginalURL,
			"custom_alias": link.CustomAlias,
			"expires_at":   link.ExpiresAt,
			"is_active":    link.IsActive,
		}

		res := tx.Model(&Link{}).Where("id = ?", link.ID).Updates(updates)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}

		return nil
	})
}
