package configloader

import (
	"os"
	"path/filepath"
	"runtime"
	"time"

	"aziz.dev/gateway/internal/config"
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
			_ = godotenv.Load("gateway/.env")
		}
	}

	serviceName := os.Getenv("SERVICE_NAME")
	servicePort := os.Getenv("SERVICE_PORT")

	shortenerServiceURL := os.Getenv("SHORTENER_SERVICE_URL")
	redirectServiceURL := os.Getenv("REDIRECT_SERVICE_URL")

	// postgres
	postgresHost := os.Getenv("POSTGRES_HOST")
	postgresUser := os.Getenv("POSTGRES_USER")
	postgresPassword := os.Getenv("POSTGRES_PASSWORD")
	postgresDB := os.Getenv("POSTGRES_DB")

	// redis
	redisAddr := os.Getenv("REDIS_ADDR")
	redisUser := os.Getenv("REDIS_USER")
	redisPassword := os.Getenv("REDIS_PASSWORD")

	// jwt
	accessSecret := os.Getenv("JWT_ACCESS_SECRET")
	refreshSecret := os.Getenv("JWT_REFRESH_SECRET")
	accessExpiry := os.Getenv("JWT_ACCESS_EXPIRY")
	refreshExpiry := os.Getenv("JWT_REFRESH_EXPIRY")

	return &config.AppConfig{
		Service: config.ServiceConfig{
			Name: serviceName,
			Port: servicePort,
			ShortenerServiceURL: shortenerServiceURL,
			RedirectServiceURL: redirectServiceURL,
		},
		Postgres: config.PostgresConfig{
			Host: postgresHost,
			User: postgresUser,
			Password: postgresPassword,
			DBName: postgresDB,
		},
		Redis: config.RedisConfig{
			Addr: redisAddr,
			Username: redisUser,
			Password: redisPassword,
		},
		JWT: config.JWTConfig{
			AccessSecret: []byte(accessSecret),
			RefreshSecret: []byte(refreshSecret),
			AccessExpiry: parseDuration(accessExpiry),
			RefreshExpiry: parseDuration(refreshExpiry),
		},
	}, nil
}

func parseDuration(d string) time.Duration {
	dur, err := time.ParseDuration(d)
	if err != nil {
		return 0
	}
	return dur
}
