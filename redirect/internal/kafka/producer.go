package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"

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
		logrus.Error("Kafka: at least one broker address is required")
		return nil, fmt.Errorf("kafka: at least one broker address is required")
	}
 
	logrus.WithField("brokers", cfg.Brokers).Info("Initializing Kafka producer")
	w := &kafka.Writer{
		Addr:         kafka.TCP(cfg.Brokers...),
		Balancer:     &kafka.Hash{},
		RequiredAcks: kafka.RequireOne,
		Async:        false,
		Transport: &kafka.Transport{
			DialTimeout: 10 * time.Second,
		},
		ErrorLogger: kafka.LoggerFunc(func(msg string, args ...any) {
			logrus.Errorf("Kafka error: "+msg, args...)
		}),
	}
 
	return &Producer{writer: w}, nil
}

func (p *Producer) Close() error {
	logrus.Info("Closing Kafka producer")
	return p.writer.Close()
}

func (p *Producer) SendEvent(ctx context.Context, topic string, message map[string]any) error {
	b, err := json.Marshal(message)
	if err != nil {
		logrus.WithError(err).Error("Failed to marshal Kafka message")
		return fmt.Errorf("kafka: marshal message: %w", err)
	}
 
	if err := p.writer.WriteMessages(ctx, kafka.Message{
		Topic: topic,
		Value: b,
	}); err != nil {
		logrus.WithError(err).WithField("topic", topic).Error("Failed to send Kafka message")
		return fmt.Errorf("kafka: write to topic %s: %w", topic, err)
	}
 
	logrus.WithField("topic", topic).Info("Successfully sent Kafka message")
	return nil
}