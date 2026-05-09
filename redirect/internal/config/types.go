package config

type AppConfig struct {
	Service ServiceConfig
	Postgres PostgresConfig
	Redis RedisConfig
	Kafka KafkaConfig
}

type ServiceConfig struct {
	Name string
	Port string
}

type PostgresConfig struct {
	Host string
	User string
	Password string
	DBName string
}

type RedisConfig struct {
	Addr string
	User string
	Password string
}

type KafkaConfig struct {
	Brokers  []string
	ConsumerTopics []string
	ProducerTopic string
	GroupID  string
}