package kafka

import (
	"context"
	"encoding/json"
	"time"

	"aziz.dev/shortener/internal/config"
	"github.com/segmentio/kafka-go"
)

type Producer struct {
	writer *kafka.Writer
}

func NewProducer(ctx context.Context, cfg *config.KafkaConfig) (*Producer, error) {
	w := &kafka.Writer{
		Addr: kafka.TCP(cfg.Brokers...),
		Balancer: &kafka.Hash{},
		RequiredAcks: kafka.RequireOne,
		Async: false,
		Transport: &kafka.Transport{
			DialTimeout: 10 * time.Second,
		},
	}

	return &Producer{writer: w}, nil
}

func (p *Producer) Close() error {
	return p.writer.Close()
}

func (p *Producer) SendEvent(ctx context.Context, topic string, message map[string]any) error {
	b, err := json.Marshal(message)
	if err != nil {
		return err
	}

	err = p.writer.WriteMessages(ctx, kafka.Message{
		Topic: topic,
		Value: b,
	})

	if err != nil {
		return err
	}

	return nil
}