package config

import "time"

type AppConfig struct {
	Service ServiceConfig
	Postgres PostgresConfig
	Redis RedisConfig
	JWT JWTConfig
}

type ServiceConfig struct {
	Name string
	Port string
	ShortenerServiceURL string
	RedirectServiceURL string
	AnalyticsServiceURL string
}

type PostgresConfig struct {
	Host string
	User string
	Password string
	DBName string
}

type RedisConfig struct {
	Addr string
	Username string
	Password string
}

type JWTConfig struct {
	AccessSecret []byte
	RefreshSecret []byte
	AccessExpiry time.Duration
	RefreshExpiry time.Duration
}