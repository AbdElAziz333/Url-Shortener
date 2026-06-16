package main

import (
	"context"
	"os"

	"aziz.dev/shortener/internal/configloader"
	"aziz.dev/shortener/internal/link"
	"aziz.dev/shortener/internal/postgres"
	"aziz.dev/shortener/internal/server"
	"github.com/sirupsen/logrus"
)

func main() {
	logrus.SetFormatter(&logrus.JSONFormatter{})
	logrus.SetOutput(os.Stdout)
	logrus.SetLevel(logrus.InfoLevel)

	logrus.Info("Starting shortener service")

	config, err := configloader.LoadFromEnv()
	if err != nil {
		logrus.WithError(err).Fatal("Failed to load config")
	}

	logrus.Info("Connecting to PostgreSQL")
	postgresDB, err := postgres.NewClient(context.Background(), &config.Postgres)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to connect to PostgreSQL")
	}

	linkRepository := link.NewRepository(postgresDB)
	cbLinkRepository := link.NewCircuitBreakerRepository(linkRepository)
	linkService := link.NewService(cbLinkRepository)
	linkHandler := link.NewHandler(linkService)

	router := server.NewRouter(linkHandler)

	go func() {
		if err := server.StartGRPC(linkService, config.Service.GRPCPort); err != nil {
			logrus.WithError(err).Fatal("Failed to start gRPC server")
		}
	}()

	logrus.Infof("Starting server on port %s", config.Service.Port)
	if err := router.Run(":" + config.Service.Port); err != nil {
		logrus.WithError(err).Fatal("Failed to start server")
	}
}