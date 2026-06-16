package configloader

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"aziz.dev/redirect/internal/config"
	"github.com/joho/godotenv"
)

func LoadFromEnv() (*config.AppConfig, error) {
	if appEnv := os.Getenv("APP_ENV"); appEnv == "local" || appEnv == "" {
		loadDotEnv()
	}

	grpcPort := os.Getenv("GRPC_PORT")
	if grpcPort == "" {
		grpcPort = "50052"
	}

	cfg := &config.AppConfig{
		Service: config.ServiceConfig{
			Name:     requireEnv("SERVICE_NAME"),
			Port:     requireEnv("SERVICE_PORT"),
			GRPCPort: grpcPort,
		},
		Postgres: config.PostgresConfig{
			Host:     requireEnv("POSTGRES_HOST"),
			User:     requireEnv("POSTGRES_USER"),
			Password: requireEnv("POSTGRES_PASSWORD"),
			DBName:   requireEnv("POSTGRES_DB"),
		},
		Redis: config.RedisConfig{
			Addr:     requireEnv("REDIS_ADDR"),
			User:     os.Getenv("REDIS_USER"),
			Password: os.Getenv("REDIS_PASSWORD"),
		},
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}

	return cfg, nil
}

func loadDotEnv() {
	_, filename, _, ok := runtime.Caller(0)
	if ok {
		envPath := filepath.Join(filepath.Dir(filename), "..", "..", ".env")
		_ = godotenv.Load(envPath)
		return
	}
	_ = godotenv.Load("redirect/.env")
}

func requireEnv(key string) string {
	return os.Getenv(key)
}
