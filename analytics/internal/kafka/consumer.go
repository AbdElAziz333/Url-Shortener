package kafka

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"aziz.dev/analytics/internal/config"
	"aziz.dev/analytics/internal/middleware"
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
	topic  string
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
		topic:  topic,
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
		start := time.Now()
		event, err := c.ReadEvent(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return // shutdown
			}
			log.Printf("kafka read error: %v", err)
			middleware.AnalyticsProcessingFailuresTotal.WithLabelValues("kafka_read_error").Inc()
			continue
		}

		err = repo.SaveClickEvent(ctx, event)
		duration := time.Since(start).Seconds()

		if err != nil {
			middleware.AnalyticsProcessingFailuresTotal.WithLabelValues("save_error").Inc()
		} else {
			middleware.AnalyticsProcessingDuration.WithLabelValues(c.topic).Observe(duration)
		}

		// Update queue lag stats
		stats := c.reader.Stats()
		middleware.AnalyticsEventsQueueDepth.WithLabelValues(c.topic).Set(float64(stats.Lag))
	}
}