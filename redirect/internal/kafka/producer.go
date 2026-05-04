package kafka

import (
	"context"
	"time"

	"github.com/segmentio/kafka-go"

	"aziz.dev/redirect/internal/config"
)

type Producer struct {
	writer *kafka.Writer
}

func NewProducer(ctx context.Context, cfg *config.KafkaConfig) (*Producer, error) {
	return &Producer{
		writer: &kafka.Writer{
			Addr: kafka.TCP(cfg.Brokers...),
			Topic: "analytics",
			Transport: &kafka.Transport{
				DialTimeout: 10 * time.Second,
			},
		}, 
	}, nil
}

func (p *Producer) Close() error {
	return p.writer.Close()
}