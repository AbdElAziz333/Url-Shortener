package main

import (
	"context"

	"aziz.dev/analytics/internal/configloader"
	"aziz.dev/analytics/internal/kafka"
	"aziz.dev/analytics/internal/mongo"
	"aziz.dev/analytics/internal/postgres"
	"aziz.dev/analytics/internal/server"
	"aziz.dev/analytics/internal/stat"
	"github.com/sirupsen/logrus"
)

func main() {
	config, err := configloader.LoadFromEnv()
	if err != nil {
		logrus.Fatalf("Failed to load config: %v", err)
	}
	logrus.Info("Config loaded successfully")

	postgresDB, err := postgres.NewClient(context.Background(), &config.Postgres)
	if err != nil {
		logrus.Fatalf("Failed to connect to Postgres: %v", err)
	}
	logrus.Info("Connected to Postgres successfully")

	mongoClient, err := mongo.NewClient(&config.Mongo)
	if err != nil {
		logrus.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	logrus.Info("Connected to MongoDB successfully")

	clickConsumer, err := kafka.NewConsumer(context.Background(), &config.Kafka, kafka.TopicRedirectsResolved)
	if err != nil {
		logrus.Fatalf("Failed to create Kafka consumer for topic %s: %v", kafka.TopicRedirectsResolved, err)
	}
	logrus.Infof("Kafka consumer created for topic: %s", kafka.TopicRedirectsResolved)

	failConsumer, err := kafka.NewConsumer(context.Background(), &config.Kafka, kafka.TopicRedirectsFailed)
	if err != nil {
		logrus.Fatalf("Failed to create Kafka consumer for topic %s: %v", kafka.TopicRedirectsFailed, err)
	}
	logrus.Infof("Kafka consumer created for topic: %s", kafka.TopicRedirectsFailed)

	statRepository := stat.NewRepository(postgresDB, mongoClient)

	go clickConsumer.Ingest(context.Background(), statRepository)
	go failConsumer.Ingest(context.Background(), statRepository)
	logrus.Info("Kafka consumers started")

	statService := stat.NewService(statRepository)
	statHandler := stat.NewHandler(statService)

	router := server.NewRouter(statHandler)

	logrus.Infof("Starting server on port %s", config.Service.Port)
	router.Run(":" + config.Service.Port)
}
