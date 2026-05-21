package config

type AppConfig struct {
	Service ServiceConfig
	Postgres PostgresConfig
	Kafka KafkaConfig
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

type KafkaConfig struct {
	Brokers  []string
	GroupID  string
}