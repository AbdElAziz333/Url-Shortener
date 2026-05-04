package stat

import (
	"context"

	"gorm.io/gorm"
)

type Repository interface {
	GetTotalClicks(ctx context.Context) ([]Dto, error)
	GetGeo(ctx context.Context) ([]Dto, error)
	GetReferrers(ctx context.Context) ([]Dto, error)
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{
		db: db,
	}
}

func (r *repository) GetTotalClicks(ctx context.Context) ([]Dto, error) {
	return nil, nil
}

func (r *repository) GetGeo(ctx context.Context) ([]Dto, error) {
	return nil, nil
}

func (r *repository) GetReferrers(ctx context.Context) ([]Dto, error) {
	return nil, nil
}