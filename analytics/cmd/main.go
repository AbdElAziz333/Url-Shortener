package main

import (
	"context"

	"aziz.dev/analytics/internal/configloader"
	"aziz.dev/analytics/internal/postgres"
	"aziz.dev/analytics/internal/mongo"
	"aziz.dev/analytics/internal/stat"
	"aziz.dev/analytics/internal/server"
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

	statRepositoy := stat.NewRepository(postgresDB, mongoClient)
	statService := stat.NewService(statRepositoy)
	statHandler := stat.NewHandler(statService)

	router := server.NewRouter(statHandler)

	router.Run(":" + config.Service.Port)
}