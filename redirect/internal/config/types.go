package config

import "fmt"

type AppConfig struct {
	Service   ServiceConfig
	Postgres  PostgresConfig
	Redis     RedisConfig
}

func (c *AppConfig) Validate() error {
	required := []struct {
		val  string
		name string
	}{
		{c.Service.Name, "SERVICE_NAME"},
		{c.Service.Port, "SERVICE_PORT"},
		{c.Postgres.Host, "POSTGRES_HOST"},
		{c.Postgres.User, "POSTGRES_USER"},
		{c.Postgres.Password, "POSTGRES_PASSWORD"},
		{c.Postgres.DBName, "POSTGRES_DB"},
		{c.Redis.Addr, "REDIS_ADDR"},
	}

	for _, r := range required {
		if r.val == "" {
			return fmt.Errorf("required environment variable %s is not set", r.name)
		}
	}

	return nil
}

type ServiceConfig struct {
	Name string
	Port string
}

type PostgresConfig struct {
	Host     string
	User     string
	Password string
	DBName   string
}

type RedisConfig struct {
	Addr     string
	User     string
	Password string
}