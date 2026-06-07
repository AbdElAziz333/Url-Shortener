package kafka

import (
	"context"
	"encoding/json"
	"time"

	"aziz.dev/analytics/internal/config"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

const (
	TopicURLsCreated         = "urls.created"
	TopicURLsDeleted         = "urls.deleted"
	TopicRedirectsRequested  = "redirects.requested"
	TopicRedirectsResolved   = "redirects.resolved"
	TopicRedirectsFailed     = "redirects.failed"
	TopicAnalyticsAggregated = "analytics.aggregated"
)

type IngestRepository interface {
	SaveClickEvent(ctx context.Context, event map[string]any) error
}

type Consumer struct {
	reader *kafka.Reader
	topic  string
}

func NewConsumer(ctx context.Context, cfg *config.KafkaConfig, topic string) (*Consumer, error) {
	logrus.WithFields(logrus.Fields{
		"brokers":  cfg.Brokers,
		"group_id": cfg.GroupID,
		"topic":    topic,
	}).Info("Creating Kafka consumer")

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        cfg.Brokers,
		GroupID:        cfg.GroupID,
		Topic:          topic,
		MinBytes:       1,
		MaxBytes:       10e6,
		CommitInterval: time.Second,
		MaxWait:        15 * time.Second,
		StartOffset:    kafka.LastOffset,
		ErrorLogger: kafka.LoggerFunc(func(msg string, args ...any) {
			logrus.Errorf("Kafka consumer error: "+msg, args...)
		}),
	})

	return &Consumer{
		reader: reader,
		topic:  topic,
	}, nil
}

func (c *Consumer) Close() error {
	logrus.Infof("Closing Kafka consumer for topic: %s", c.topic)
	return c.reader.Close()
}

func (c *Consumer) ReadEvent(ctx context.Context) (map[string]any, error) {
	msg, err := c.reader.ReadMessage(ctx)
	if err != nil {
		return nil, err
	}

	var event map[string]any
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		logrus.WithError(err).WithField("raw_message", string(msg.Value)).Error("Failed to unmarshal Kafka event")
		return nil, err
	}

	return event, nil
}

func (c *Consumer) Ingest(ctx context.Context, repo IngestRepository) {
	logrus.Infof("Starting ingestion for topic: %s", c.topic)

	// Add initial backoff to let Kafka group coordinator fully initialize
	logrus.Infof("Waiting 10 seconds for Kafka group coordinator to become available for topic: %s", c.topic)
	select {
	case <-time.After(10 * time.Second):
	case <-ctx.Done():
		logrus.Info("Ingestion cancelled during initial backoff")
		return
	}

	for {
		select {
		case <-ctx.Done():
			logrus.Info("Ingestion stopped due to context cancellation")
			return
		default:
		}

		event, err := c.ReadEvent(ctx)
		if err != nil {
			if ctx.Err() != nil {
				logrus.Info("Ingestion stopped due to context cancellation")
				return
			}

			logrus.WithError(err).WithFields(logrus.Fields{
				"topic": c.topic,
			}).Warn("Kafka read error, will retry in 5 seconds")

			// Backoff before retrying
			select {
			case <-time.After(5 * time.Second):
			case <-ctx.Done():
			}
			continue
		}

		err = repo.SaveClickEvent(ctx, event)

		if err != nil {
			logrus.WithError(err).WithField("topic", c.topic).Error("Failed to save click event")
		} else {
			logrus.WithFields(logrus.Fields{
				"topic": c.topic,
			}).Debug("Successfully processed event")
		}
	}
}
