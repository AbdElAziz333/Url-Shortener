package configloader

import (
	"os"
	"path/filepath"
	"runtime"

	"aziz.dev/shortener/internal/config"
	"github.com/joho/godotenv"
)

func LoadFromEnv() (*config.AppConfig, error) {
	appEnv := os.Getenv("APP_ENV")
	if appEnv == "local" || appEnv == "" {
		_, filename, _, ok := runtime.Caller(0)
		if ok {
			dir := filepath.Dir(filename)
			envPath := filepath.Join(dir, "..", "..", ".env")
			_ = godotenv.Load(envPath)
		} else {
			_ = godotenv.Load("shortener/.env")
		}
	}

	return &config.AppConfig{
		Service: config.ServiceConfig{
			Name: os.Getenv("SERVICE_NAME"),
			Port: os.Getenv("SERVICE_PORT"),
		},
		Postgres: config.PostgresConfig{
			Host:     os.Getenv("POSTGRES_HOST"),
			User:     os.Getenv("POSTGRES_USER"),
			Password: os.Getenv("POSTGRES_PASSWORD"),
			DBName:   os.Getenv("POSTGRES_DB"),
		},
	}, nil
}
