package stat

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"gorm.io/gorm"
)

type Repository interface {
	GetTotalClicks(ctx context.Context) ([]Dto, error)
	GetGeo(ctx context.Context) ([]Dto, error)
	GetReferrers(ctx context.Context) ([]Dto, error)
}

type repository struct {
	db *gorm.DB
	mongoClient *mongo.Client
}

func NewRepository(db *gorm.DB, mongoClient *mongo.Client) Repository {
	return &repository{
		db: db,
		mongoClient: mongoClient,
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