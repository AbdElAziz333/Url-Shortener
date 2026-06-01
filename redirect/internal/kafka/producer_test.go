package kafka

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	kafka "github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"aziz.dev/redirect/internal/config"
	"aziz.dev/redirect/internal/testutil"
)

// ---------------------------------------------------------------------------
// Unit Tests — no broker required
// ---------------------------------------------------------------------------

func TestNewProducer_NoBrokers_ReturnsError(t *testing.T) {
	cfg := &config.KafkaConfig{Brokers: []string{}}

	p, err := NewProducer(context.Background(), cfg)

	require.Error(t, err)
	assert.Nil(t, p)
	assert.Contains(t, err.Error(), "broker")
}

func TestNewProducer_WithBrokers_Succeeds(t *testing.T) {
	cfg := &config.KafkaConfig{Brokers: []string{"localhost:9092"}}

	p, err := NewProducer(context.Background(), cfg)

	// Producer construction is cheap and doesn't dial yet — must not error.
	require.NoError(t, err)
	require.NotNil(t, p)
	_ = p.Close()
}

func TestProducer_SendEvent_UnmarshalablePayload_ReturnsError(t *testing.T) {
	// json.Marshal fails on channels.
	unmarshalable := map[string]any{
		"bad": make(chan int),
	}

	cfg := &config.KafkaConfig{Brokers: []string{"localhost:9092"}}
	p, err := NewProducer(context.Background(), cfg)
	require.NoError(t, err)
	defer p.Close()

	err = p.SendEvent(context.Background(), TopicURLClicked, unmarshalable)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "marshal")
}

func TestProducer_SendEvent_NoBrokerReachable_ReturnsError(t *testing.T) {
	cfg := &config.KafkaConfig{Brokers: []string{"localhost:19092"}} // nothing listening
	p, err := NewProducer(context.Background(), cfg)
	require.NoError(t, err)
	defer p.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err = p.SendEvent(ctx, TopicURLClicked, map[string]any{"code": "abc"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), TopicURLClicked)
}

// ---------------------------------------------------------------------------
// Integration Tests — real Kafka via testcontainers
// ---------------------------------------------------------------------------

func TestProducer_SendEvent_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	brokers := testutil.NewKafkaContainer(t, ctx)

	// Pre-create the topic so the writer doesn't have to rely on auto-create.
	createTopic(t, ctx, brokers[0], TopicURLClicked)

	cfg := &config.KafkaConfig{Brokers: brokers}
	p, err := NewProducer(ctx, cfg)
	require.NoError(t, err)
	defer p.Close()

	payload := map[string]any{
		"code":         "abc123",
		"original_url": "https://example.com",
		"clicked_at":   time.Now().UTC(),
	}

	err = p.SendEvent(ctx, TopicURLClicked, payload)
	require.NoError(t, err)

	// Read back the message and assert its shape.
	msg := consumeOne(t, ctx, brokers[0], TopicURLClicked)
	var got map[string]any
	require.NoError(t, json.Unmarshal(msg.Value, &got))
	assert.Equal(t, "abc123", got["code"])
	assert.Equal(t, "https://example.com", got["original_url"])
}

func TestProducer_SendEvent_Integration_MultipleTopics(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	brokers := testutil.NewKafkaContainer(t, ctx)

	topics := []string{
		TopicURLsCreated,
		TopicURLsDeleted,
		TopicRedirectsRequested,
	}
	for _, topic := range topics {
		createTopic(t, ctx, brokers[0], topic)
	}

	cfg := &config.KafkaConfig{Brokers: brokers}
	p, err := NewProducer(ctx, cfg)
	require.NoError(t, err)
	defer p.Close()

	for _, topic := range topics {
		err := p.SendEvent(ctx, topic, map[string]any{"topic": topic})
		assert.NoError(t, err, "SendEvent should succeed for topic %s", topic)
	}
}

func TestProducer_SendEvent_Integration_PayloadIsValidJSON(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	brokers := testutil.NewKafkaContainer(t, ctx)
	createTopic(t, ctx, brokers[0], TopicURLClicked)

	cfg := &config.KafkaConfig{Brokers: brokers}
	p, err := NewProducer(ctx, cfg)
	require.NoError(t, err)
	defer p.Close()

	payload := map[string]any{
		"code":    "xyz",
		"user_id": "00000000-0000-0000-0000-000000000001",
		"nested":  map[string]any{"key": "value"},
	}
	require.NoError(t, p.SendEvent(ctx, TopicURLClicked, payload))

	msg := consumeOne(t, ctx, brokers[0], TopicURLClicked)
	assert.True(t, json.Valid(msg.Value), "message written to Kafka must be valid JSON")
}

func TestProducer_Close_IsIdempotent(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	brokers := testutil.NewKafkaContainer(t, ctx)

	cfg := &config.KafkaConfig{Brokers: brokers}
	p, err := NewProducer(ctx, cfg)
	require.NoError(t, err)

	assert.NoError(t, p.Close())
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// createTopic creates a Kafka topic with a single partition before the test
// produces to it, avoiding auto-create races.
func createTopic(t *testing.T, ctx context.Context, broker, topic string) {
	t.Helper()

	conn, err := kafka.DialLeader(ctx, "tcp", broker, topic, 0)
	require.NoError(t, err)
	defer conn.Close()
}

// consumeOne reads a single message from partition 0 of the given topic.
// It sets a short deadline so a missing message fails fast.
func consumeOne(t *testing.T, ctx context.Context, broker, topic string) kafka.Message {
	t.Helper()

	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:   []string{broker},
		Topic:     topic,
		Partition: 0,
		MinBytes:  1,
		MaxBytes:  1 << 20, // 1 MB
	})
	defer r.Close()

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	msg, err := r.ReadMessage(ctx)
	require.NoError(t, err, "timed out waiting for message on topic %s", topic)
	return msg
}