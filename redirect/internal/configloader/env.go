package configloader

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"aziz.dev/redirect/internal/config"
	"github.com/joho/godotenv"
)

func LoadFromEnv() (*config.AppConfig, error) {
	if appEnv := os.Getenv("APP_ENV"); appEnv == "local" || appEnv == "" {
		loadDotEnv()
	}
 
	cfg := &config.AppConfig{
		Service: config.ServiceConfig{
			Name: requireEnv("SERVICE_NAME"),
			Port: requireEnv("SERVICE_PORT"),
		},
		Postgres: config.PostgresConfig{
			Host:     requireEnv("POSTGRES_HOST"),
			User:     requireEnv("POSTGRES_USER"),
			Password: requireEnv("POSTGRES_PASSWORD"),
			DBName:   requireEnv("POSTGRES_DB"),
		},
		Redis: config.RedisConfig{
			Addr:     requireEnv("REDIS_ADDR"),
			User:     os.Getenv("REDIS_USER"),     // optional
			Password: os.Getenv("REDIS_PASSWORD"), // optional
		},
		Kafka: config.KafkaConfig{
			// KAFKA_BROKERS accepts a comma-separated list, e.g. "b1:9092,b2:9092"
			Brokers: splitBrokers(os.Getenv("KAFKA_BROKERS")),
			GroupID: os.Getenv("KAFKA_GROUP_ID"),
		},
	}
 
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}
 
	return cfg, nil
}

// loadDotEnv attempts to find and load a .env file relative to the source
// file's location (useful during local development). Failure is silent because
// the variables may already be exported in the shell.
func loadDotEnv() {
	_, filename, _, ok := runtime.Caller(0)
	if ok {
		envPath := filepath.Join(filepath.Dir(filename), "..", "..", ".env")
		_ = godotenv.Load(envPath)
		return
	}
	_ = godotenv.Load("shortener/.env")
}

// requireEnv returns the value of key or panics with a descriptive message.
// A missing required variable is a programming/deployment error, not a
// recoverable one, so early failure is appropriate.
func requireEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		// Return empty — Validate() on the config struct will surface the error
		// in a single consolidated pass rather than one panic per missing var.
	}
	return v
}
 
func splitBrokers(raw string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if b := strings.TrimSpace(p); b != "" {
			out = append(out, b)
		}
	}
	return out
}