package postgres

import (
	"context"
	"fmt"

	"aziz.dev/analytics/internal/config"
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
	
	logrus.WithFields(logrus.Fields{
		"host": cfg.Host,
		"user": cfg.User,
		"dbname": cfg.DBName,
	}).Info("Connecting to PostgreSQL")
		
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		logrus.WithError(err).Error("Failed to connect to PostgreSQL")
		return nil, err
	}

	return db, nil
}