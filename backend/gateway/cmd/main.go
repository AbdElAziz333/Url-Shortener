package main

import (
	"context"
	"os"
	// "os/signal"
	// "syscall"

	"aziz.dev/gateway/internal/configloader"
	"aziz.dev/gateway/internal/grpcclient"
	"aziz.dev/gateway/internal/middleware"
	"aziz.dev/gateway/internal/postgres"
	"aziz.dev/gateway/internal/redis"
	"aziz.dev/gateway/internal/security"
	"aziz.dev/gateway/internal/server"
	"aziz.dev/gateway/internal/user"
	"github.com/sirupsen/logrus"
)

func main() {
	logrus.SetFormatter(&logrus.JSONFormatter{})
	logrus.SetOutput(os.Stdout)
	logrus.SetLevel(logrus.InfoLevel)

	logrus.Info("Starting gateway service...")

	config, err := configloader.LoadFromEnv()
	if err != nil {
		logrus.WithError(err).Fatal("Failed to load config")
	}

	logrus.Info("Connecting to PostgreSQL...")
	postgresDB, err := postgres.NewClient(context.Background(), &config.Postgres)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to connect to PostgreSQL")
	}
	logrus.Info("PostgreSQL connected successfully")

	logrus.Info("Connecting to Redis...")
	redisClient, err := redis.NewClient(context.Background(), &config.Redis)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to connect to Redis")
	}
	logrus.Info("Redis connected successfully")

	userRepository := user.NewRepository(postgresDB)
	jwtService, err := security.NewService(&config.JWT, redisClient)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to initialize JWT service")
	}
	
	userService := user.NewService(userRepository, redisClient, jwtService)
	userHandler := user.NewHandler(userService)

	grpcClients, err := grpcclient.NewClients(config.Service.ShortenerServiceURL, config.Service.RedirectServiceURL)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to initialize gRPC clients")
	}
	defer grpcClients.Close()

	grpcHandler := grpcclient.NewHandler(grpcClients)
	router := server.NewRouter(userHandler)

	shortener := router.Group("/shortener")
	shortener.GET("/api/links", middleware.AccessTokenMiddleware(&config.JWT), grpcHandler.GetAllLinks)
	shortener.POST("/api/links", middleware.AccessTokenMiddleware(&config.JWT), grpcHandler.CreateLink)
	shortener.PATCH("/api/links/:code/expiry", middleware.AccessTokenMiddleware(&config.JWT), grpcHandler.UpdateExpiry)
	shortener.DELETE("/api/links/:code", middleware.AccessTokenMiddleware(&config.JWT), grpcHandler.DeleteLink)

	redirect := router.Group("/redirect")
	redirect.GET("/:code", grpcHandler.ResolveCode)

	logrus.Infof("Gateway service listening on port %s", config.Service.Port)
	router.Run(":"+config.Service.Port)
}

// func GracefulShutdown() {
// 	sigChan := make(chan os.Signal, 1)
// 	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
// 	<-sigChan
// 	logrus.Info("Shutting down gateway service gracefully...")
// }