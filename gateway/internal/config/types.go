package config

type PostgresConfig struct {
	Host string
	User string
	Password string
	DBName string
}

type RedisConfig struct {
	Host string
	User string
	Password string
}