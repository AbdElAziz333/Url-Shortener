package main

import (
	"context"

	"aziz.dev/redirect/internal/configloader"
	"aziz.dev/redirect/internal/kafka"
	"aziz.dev/redirect/internal/postgres"
	"aziz.dev/redirect/internal/redis"
	"aziz.dev/redirect/internal/resolve"
	"aziz.dev/redirect/internal/server"
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

	redisClient, err := redis.NewClient(context.Background(), &config.Redis)
	if err != nil {
		panic(err)
	}

	kafkaProducer, err := kafka.NewProducer(context.Background(), &config.Kafka)
	if err != nil {
		panic(err)
	}
	
	resolveRepository := resolve.NewRepository(postgresDB)
	resolveService := resolve.NewService(resolveRepository, redisClient, kafkaProducer)
	resolveHandler := resolve.NewHandler(resolveService)

	router := server.NewRouter(resolveHandler)

	router.Run(":"+config.Service.Port)
}