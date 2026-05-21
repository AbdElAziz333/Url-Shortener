package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"

	"aziz.dev/redirect/internal/config"
)

const (
	TopicURLsCreated         = "urls.created"
	TopicURLsDeleted         = "urls.deleted"
	TopicRedirectsRequested  = "redirects.requested"
	TopicRedirectsResolved   = "redirects.resolved"
	TopicRedirectsFailed     = "redirects.failed"
	TopicURLClicked          = "url-clicked"
)

type Producer struct {
	writer *kafka.Writer
}

func NewProducer(_ context.Context, cfg *config.KafkaConfig) (*Producer, error) {
	if len(cfg.Brokers) == 0 {
		return nil, fmt.Errorf("kafka: at least one broker address is required")
	}
 
	w := &kafka.Writer{
		Addr:         kafka.TCP(cfg.Brokers...),
		Balancer:     &kafka.Hash{},
		RequiredAcks: kafka.RequireOne,
		Async:        false,
		Transport: &kafka.Transport{
			DialTimeout: 10 * time.Second,
		},
		// Propagate write errors to the caller rather than silently dropping.
		ErrorLogger: kafka.LoggerFunc(func(msg string, args ...any) {
			// Integrate with your structured logger here if desired.
			_ = fmt.Sprintf(msg, args...)
		}),
	}
 
	return &Producer{writer: w}, nil
}

func (p *Producer) Close() error {
	return p.writer.Close()
}

func (p *Producer) SendEvent(ctx context.Context, topic string, message map[string]any) error {
	b, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("kafka: marshal message: %w", err)
	}
 
	if err := p.writer.WriteMessages(ctx, kafka.Message{
		Topic: topic,
		Value: b,
	}); err != nil {
		return fmt.Errorf("kafka: write to topic %s: %w", topic, err)
	}
 
	return nil
}