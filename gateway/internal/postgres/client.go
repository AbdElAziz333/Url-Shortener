package postgres

import (
	"fmt"

	"aziz.dev/gateway/internal/config"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func NewClient(serviceName string, cfg config.PostgresConfig) (*gorm.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=5432 sslmode=disable",
		cfg.Host,
		cfg.User,
		cfg.Password,
		cfg.DBName,
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	return db, nil
}