package resolve

import (
	"context"

	"gorm.io/gorm"
)

type Repository interface {
	Find(ctx context.Context, code string) (*Link, error)
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{
		db: db,
	}
}

func (r *repository) Find(ctx context.Context, code string) (*Link, error) {
	var link Link

	err := r.db.WithContext(ctx).Where("code = ?", code).First(&link).Error
	if err != nil {
		return nil, err
	}

	return &link, nil
}