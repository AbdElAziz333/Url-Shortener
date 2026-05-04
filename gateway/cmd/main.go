package main

import (
	"context"

	"aziz.dev/gateway/internal/configloader"
	"aziz.dev/gateway/internal/postgres"
	"aziz.dev/gateway/internal/proxy"

	// "aziz.dev/gateway/internal/proxy"
	"aziz.dev/gateway/internal/redis"
	"aziz.dev/gateway/internal/server"
	"aziz.dev/gateway/internal/user"
	"aziz.dev/gateway/internal/jwt"
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

	userRepository := user.NewRepository(postgresDB)
	jwtService := jwt.NewService(redisClient)
	userService := user.NewService(userRepository, redisClient, jwtService)
	userHandler := user.NewHandler(userService)

	router := server.NewRouter(userHandler)

	shortener := router.Group("/api/links")
	shortener.POST("/", proxy.ReverseProxy("/api/links", "/api/links", config.Service.ShortenerServiceURL))
	shortener.GET("/", proxy.ReverseProxy("/api/links", "/api/links", config.Service.ShortenerServiceURL))
	shortener.PATCH("/:code", proxy.ReverseProxy("/api/links", "/api/links", config.Service.ShortenerServiceURL))
	shortener.DELETE("/:code", proxy.ReverseProxy("/api/links", "/api/links", config.Service.ShortenerServiceURL))

	redirect := router.Group("/r")
	redirect.GET("/:code", proxy.ReverseProxy("/r", "/r", config.Service.RedirectServiceURL))

	analytics := router.Group("/api/analytics")
	analytics.GET("/stats/:code", proxy.ReverseProxy("/api/analytics", "/api/analytics", config.Service.AnalyticsServiceURL))
	analytics.GET("/stats/:code/geo", proxy.ReverseProxy("/api/analytics", "/api/analytics", config.Service.AnalyticsServiceURL))
	analytics.GET("/stats/:code/referrers", proxy.ReverseProxy("/api/analytics", "/api/analytics", config.Service.AnalyticsServiceURL))

	router.Run(":"+config.Service.Port)
}

func GracefulShutdown() {

}