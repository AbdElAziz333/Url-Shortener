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
	w := &kafka.Writer{
		Addr: kafka.TCP(cfg.Brokers...),
		Topic: cfg.ProducerTopic,
		Balancer: &kafka.Hash{},
		RequiredAcks: kafka.RequireOne,
		Async: false,
		Transport: &kafka.Transport{
			DialTimeout: 10 * time.Second,
		},
	}

	return &Producer{writer: w}, nil
}

func (p *Producer) Publish(ctx context.Context, key, value []byte) error {
    return p.writer.WriteMessages(ctx, kafka.Message{  // add an actual Publish method
        Key:   key,   // e.g. short_code bytes — ensures ordering per URL
        Value: value, // JSON-encoded event
    })
}

func (p *Producer) Close() error {
	return p.writer.Close()
}