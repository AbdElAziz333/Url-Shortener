package main

import (
	"context"

	"aziz.dev/shortener/internal/link"
	"aziz.dev/shortener/internal/middleware"
	"aziz.dev/shortener/internal/server"
	"aziz.dev/shortener/internal/configloader"
	"aziz.dev/shortener/internal/postgres"
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
	middleware.StartDBStatsTracking(postgresDB, "shortener")

	linkRepository := link.NewRepository(postgresDB)
	linkService := link.NewService(linkRepository)
	linkHandler := link.NewHandler(linkService)

	router := server.NewRouter(linkHandler)

	router.Run(":"+config.Service.Port)
}