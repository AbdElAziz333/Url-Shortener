package kafka

import (
	"context"
	"time"

	"aziz.dev/analytics/internal/config"
	"github.com/segmentio/kafka-go"
)

type Consumer struct {
	reader *kafka.Reader
}

func NewConsumer(ctx context.Context, cfg *config.KafkaConfig) (*Consumer, error) {
	// segmantio/kafka-go reader only supports one topic at a time.
	// for multiple topics, either:
	// A) run one reader goroutine per topic, or
	// B) use a consumer group coordinator externally
	// this is the single-reader version -- wire one per topic upstream
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: cfg.Brokers,
		GroupID: cfg.GroupID,
		Topic:   cfg.ConsumerTopics[0],
		MinBytes: 1,
		MaxBytes: 10e6,
		CommitInterval: time.Second,
		StartOffset: kafka.LastOffset,
	})
	
	return &Consumer{
		reader: reader,
	}, nil
}

func (c *Consumer) ReadMessage(ctx context.Context) (kafka.Message, error) {
	return c.reader.ReadMessage(ctx)
}

func (c *Consumer) Close() error {
	return c.reader.Close()
}
