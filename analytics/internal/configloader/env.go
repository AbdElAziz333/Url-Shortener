package configloader

import (
	"os"
	"path/filepath"
	"runtime"

	"aziz.dev/analytics/internal/config"
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
			_ = godotenv.Load("analytics/.env")
		}
	}

	serviceName := os.Getenv("SERVICE_NAME")
	servicePort := os.Getenv("SERVICE_PORT")

	// postgres
	postgresHost := os.Getenv("POSTGRES_HOST")
	postgresUser := os.Getenv("POSTGRES_USER")
	postgresPassword := os.Getenv("POSTGRES_PASSWORD")
	postgresDB := os.Getenv("POSTGRES_DB")

	// mongo
	mongoURI := os.Getenv("MONGO_URI")

	// kafka
	kafkaBrokers := os.Getenv("KAFKA_BROKERS")
	kafkaGroupID := os.Getenv("KAFKA_GROUP_ID")

	return &config.AppConfig{
		Service: config.ServiceConfig{
			Name: serviceName,
			Port: servicePort,
		},
		Postgres: config.PostgresConfig{
			Host:     postgresHost,
			User:     postgresUser,
			Password: postgresPassword,
			DBName:   postgresDB,
		},
		Mongo: config.MongoConfig{
			URI: mongoURI,
		},
		Kafka: config.KafkaConfig{
			Brokers:  []string{kafkaBrokers},
			GroupID: kafkaGroupID,
		},
	}, nil
}
