package config

type AppConfig struct {
	Service  ServiceConfig
	Postgres PostgresConfig
}

type ServiceConfig struct {
	Name     string
	Port     string
	GRPCPort string
}

type PostgresConfig struct {
	Host     string
	User     string
	Password string
	DBName   string
}
