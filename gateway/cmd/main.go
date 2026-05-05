package main

import (
	"context"

	"aziz.dev/gateway/internal/configloader"
	"aziz.dev/gateway/internal/middleware"
	"aziz.dev/gateway/internal/postgres"
	"aziz.dev/gateway/internal/proxy"

	// "aziz.dev/gateway/internal/proxy"
	"aziz.dev/gateway/internal/redis"
	"aziz.dev/gateway/internal/security"
	"aziz.dev/gateway/internal/server"
	"aziz.dev/gateway/internal/user"
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
	jwtService, err := security.NewService(&config.JWT, redisClient)
	if err != nil {
		panic(err)
	}
	
	userService := user.NewService(userRepository, redisClient, jwtService)
	userHandler := user.NewHandler(userService)

	router := server.NewRouter(userHandler)

	shortener := router.Group("/shortener")
	shortener.POST("/links", middleware.AccessTokenMiddleware(&config.JWT), proxy.ReverseProxy("/shortener", "/shortener", config.Service.ShortenerServiceURL))
	shortener.GET("/links", middleware.AccessTokenMiddleware(&config.JWT), proxy.ReverseProxy("/shortener", "/shortener", config.Service.ShortenerServiceURL))
	shortener.PATCH("/links/:code", middleware.AccessTokenMiddleware(&config.JWT), proxy.ReverseProxy("/shortener", "/shortener", config.Service.ShortenerServiceURL))
	shortener.DELETE("/links/:code", middleware.AccessTokenMiddleware(&config.JWT), proxy.ReverseProxy("/shortener", "/shortener", config.Service.ShortenerServiceURL))

	redirect := router.Group("/redirect")
	redirect.GET("/:code", proxy.ReverseProxy("/redirect", "/redirect", config.Service.RedirectServiceURL))

	analytics := router.Group("/analytics")
	analytics.GET("/api/stats/:code", middleware.AccessTokenMiddleware(&config.JWT), proxy.ReverseProxy("/analytics", "/analytics", config.Service.AnalyticsServiceURL))
	analytics.GET("/api/stats/:code/geo", middleware.AccessTokenMiddleware(&config.JWT), proxy.ReverseProxy("/analytics", "/analytics", config.Service.AnalyticsServiceURL))
	analytics.GET("/api/stats/:code/referrers", middleware.AccessTokenMiddleware(&config.JWT), proxy.ReverseProxy("/analytics", "/analytics", config.Service.AnalyticsServiceURL))

	router.Run(":"+config.Service.Port)
}

func GracefulShutdown() {

}