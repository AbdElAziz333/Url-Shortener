package main

import (
	"context"

	"aziz.dev/analytics/internal/configloader"
	"aziz.dev/analytics/internal/kafka"
	"aziz.dev/analytics/internal/mongo"
	"aziz.dev/analytics/internal/postgres"
	"aziz.dev/analytics/internal/server"
	"aziz.dev/analytics/internal/stat"
)

func main() {
	config, err := configloader.LoadFromEnv()
	if err != nil {
		panic(err)
	}

	postgresDB, err := postgres.NewClient(context.Background(), &config.Postgres)
	if err != nil {
		panic(err)
	}

	mongoClient, err := mongo.NewClient(&config.Mongo)
	if err != nil {
		panic(err)
	}

	clickConsumer, err := kafka.NewConsumer(context.Background(), &config.Kafka, kafka.TopicRedirectsResolved)
	if err != nil {
		panic(err)
	}

	failConsumer, err := kafka.NewConsumer(context.Background(), &config.Kafka, kafka.TopicRedirectsFailed)
	if err != nil {
		panic(err)
	}

	statRepository := stat.NewRepository(postgresDB, mongoClient)

	go clickConsumer.Ingest(context.Background(), statRepository)
	go failConsumer.Ingest(context.Background(), statRepository)

	statService := stat.NewService(statRepository)
	statHandler := stat.NewHandler(statService)

	router := server.NewRouter(statHandler)

	router.Run(":" + config.Service.Port)
}
