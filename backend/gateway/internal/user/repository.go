package user

import (
	"context"
	"errors"

	"gorm.io/gorm"
)

type Repository interface {
	FindByEmail(ctx context.Context, email string) (*User, error)
	Create(ctx context.Context, user User) error
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{
		db: db,
	}
}

func (r *repository) FindByEmail(ctx context.Context, email string) (*User, error) {
	var user User
	result := r.db.WithContext(ctx).Where("email = ?", email).First(&user)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, result.Error
	}

	return &user, nil
}

func (r *repository) Create(ctx context.Context, user User) error {
	result := r.db.WithContext(ctx).Create(&user)
	if result.Error != nil {
		return result.Error
	}

	return nil
}