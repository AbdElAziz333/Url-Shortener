package postgres

import (
	"context"
	"fmt"

	"aziz.dev/gateway/internal/config"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func NewClient(ctx context.Context, cfg *config.PostgresConfig) (*gorm.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=5432 sslmode=disable",
		cfg.Host,
		cfg.User,
		cfg.Password,
		cfg.DBName,
	)
		
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		logrus.WithError(err).Error("Failed to open PostgreSQL connection")
		return nil, err
	}

	logrus.Info("PostgreSQL client initialized")
	return db, nil
}