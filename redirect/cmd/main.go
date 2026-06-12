package main

import (
	"context"

	"aziz.dev/redirect/internal/configloader"
	"aziz.dev/redirect/internal/postgres"
	"aziz.dev/redirect/internal/redis"
	"aziz.dev/redirect/internal/resolve"
	"aziz.dev/redirect/internal/server"
	"github.com/sirupsen/logrus"
)

func main() {
	logrus.Info("Starting redirect service")
	config, err := configloader.LoadFromEnv()
	if err != nil {
		logrus.WithError(err).Fatal("Failed to load config")
	}

	logrus.WithField("host", config.Postgres.Host).Info("Initializing Postgres client")
	postgresDB, err := postgres.NewClient(context.Background(), &config.Postgres)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to initialize Postgres client")
	}

	logrus.WithField("addr", config.Redis.Addr).Info("Initializing Redis client")
	redisClient, err := redis.NewClient(context.Background(), &config.Redis)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to initialize Redis client")
	}

	resolveRepository := resolve.NewRepository(postgresDB)
	cbResolveRepository := resolve.NewCircuitBreakerRepository(resolveRepository)
	resolveService := resolve.NewService(cbResolveRepository, redisClient)
	resolveHandler := resolve.NewHandler(resolveService)

	router := server.NewRouter(resolveHandler)

	logrus.WithField("port", config.Service.Port).Info("Starting server")
	router.Run(":" + config.Service.Port)
}
