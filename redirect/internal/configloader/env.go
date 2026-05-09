package configloader

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"aziz.dev/redirect/internal/config"
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

	serviceName := os.Getenv("SERVICE_NAME")
	servicePort := os.Getenv("SERVICE_PORT")

	// postgres
	postgresHost := os.Getenv("POSTGRES_HOST")
	postgresUser := os.Getenv("POSTGRES_USER")
	postgresPassword := os.Getenv("POSTGRES_PASSWORD")
	postgresDB := os.Getenv("POSTGRES_DB")

	// redis
	redisAddr := os.Getenv("REDIS_ADDR")
	redisUser := os.Getenv("REDIS_USER")
	redisPassword := os.Getenv("REDIS_PASSWORD")

	//kafka
	kafkaBrokers := os.Getenv("KAFKA_BROKERS")
	kafkaProducerTopic := os.Getenv("KAFKA_PRODUCER_TOPIC")
	kafkaGroupID := os.Getenv("KAFKA_GROUP_ID")

	// comma-separated topics from single env var
	consumerTopicsRaw := os.Getenv("KAFKA_CONSUMER_TOPICS")
	var kafkaConsumerTopics []string

	for _, t := range strings.Split(consumerTopicsRaw, ",") {
		if t = strings.TrimSpace(t); t != "" {
			kafkaConsumerTopics = append(kafkaConsumerTopics, t)
		}
	}

	return &config.AppConfig{
		Service: config.ServiceConfig{
			Name: serviceName,
			Port: servicePort,
		},
		Postgres: config.PostgresConfig{
			Host: postgresHost,
			User: postgresUser,
			Password: postgresPassword,
			DBName: postgresDB,
		},
		Redis: config.RedisConfig{
			Addr: redisAddr,
			User: redisUser,
			Password: redisPassword,
		},
		Kafka: config.KafkaConfig{
			Brokers:  []string{kafkaBrokers},
			ConsumerTopics: kafkaConsumerTopics,
			ProducerTopic: kafkaProducerTopic,
			GroupID: kafkaGroupID,
		},
	}, nil
}