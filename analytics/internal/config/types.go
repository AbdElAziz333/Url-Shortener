package config

type AppConfig struct {
	Service  ServiceConfig
	Postgres PostgresConfig
	Kafka    KafkaConfig
	Mongo    MongoConfig
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

type MongoConfig struct {
	URI string
}

type KafkaConfig struct {
	Brokers  []string
	ConsumerTopics []string
	ProducerTopic string
	GroupID  string
}