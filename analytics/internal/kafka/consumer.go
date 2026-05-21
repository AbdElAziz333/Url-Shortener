package kafka

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"aziz.dev/analytics/internal/config"
	"github.com/segmentio/kafka-go"
)

const (
	TopicURLsCreated = "urls.created"
	TopicURLsDeleted = "urls.deleted"
	TopicRedirectsRequested = "redirects.requested"
	TopicRedirectsResolved = "redirects.resolved"
	TopicRedirectsFailed = "redirects.failed"
	TopicAnalyticsAggregated = "analytics.aggregated"
)

type IngestRepository interface {
	SaveClickEvent(ctx context.Context, event map[string]any) error
}

type Consumer struct {
	reader *kafka.Reader
}

func NewConsumer(ctx context.Context, cfg *config.KafkaConfig, topic string) (*Consumer, error) {
	// segmantio/kafka-go reader only supports one topic at a time.
	// for multiple topics, either:
	// A) run one reader goroutine per topic, or
	// B) use a consumer group coordinator externally
	// this is the single-reader version -- wire one per topic upstream
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: cfg.Brokers,
		GroupID: cfg.GroupID,
		Topic:   topic,
		MinBytes: 1,
		MaxBytes: 10e6,
		CommitInterval: time.Second,
		MaxWait: 15 * time.Second,
	})
	
	return &Consumer{
		reader: reader,
	}, nil
}

func (c *Consumer) Close() error {
	return c.reader.Close()
}

func (c *Consumer) ReadEvent(ctx context.Context) (map[string]any, error) {
	msg, err := c.reader.ReadMessage(ctx)
	if err != nil {
		return nil, err
	}

	var event map[string]any
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		return nil, err
	}

	return event, nil
}

func (c *Consumer) Ingest(ctx context.Context, repo IngestRepository) {
    for {
        event, err := c.ReadEvent(ctx)
        if err != nil {
            if ctx.Err() != nil { return } // shutdown
            log.Printf("kafka read error: %v", err)
            continue
        }
        repo.SaveClickEvent(ctx, event)
    }
}