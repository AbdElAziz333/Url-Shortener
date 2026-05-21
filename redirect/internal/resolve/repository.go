package resolve

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"
)

var ErrNotFound = errors.New("link not found")

type Repository interface {
	Find(ctx context.Context, code string) (*Link, error)
}
 
type repository struct {
	db *gorm.DB
}
 
func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) Find(ctx context.Context, code string) (*Link, error) {
	var link Link
 
	err := r.db.WithContext(ctx).Where("code = ?", code).First(&link).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("%w: %s", ErrNotFound, code)
		}
		return nil, err
	}
 
	return &link, nil
}