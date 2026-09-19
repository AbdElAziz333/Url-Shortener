package postgres

import (
	"context"
	"fmt"

	"aziz.dev/gateway/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sirupsen/logrus"
)

func NewClient(ctx context.Context, cfg *config.PostgresConfig) (*pgxpool.Pool, error) {
	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:5432/%s?sslmode=%s",
		cfg.User,
		cfg.Password,
		cfg.Host,
		cfg.DBName,
		"disable",
	)

	logrus.WithFields(logrus.Fields{
		"host":   cfg.Host,
		"dbname": cfg.DBName,
	}).Info("Connecting to Postgres")

	db, err := pgxpool.New(ctx, dsn)
	if err != nil {
		logrus.WithError(err).Error("Failed to connect to Postgres")
		return nil, err
	}

	logrus.Info("Successfully connected to Postgres")
	return db, nil
}