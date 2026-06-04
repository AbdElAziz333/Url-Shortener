package postgres

import (
	"context"
	"fmt"

	"aziz.dev/redirect/internal/config"
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
		"host":   cfg.Host,
		"dbname": cfg.DBName,
	}).Info("Connecting to Postgres")
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		logrus.WithError(err).Error("Failed to connect to Postgres")
		return nil, err
	}

	logrus.Info("Successfully connected to Postgres")
	return db, nil
}