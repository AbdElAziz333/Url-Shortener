package kafka

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"aziz.dev/shortener/internal/config"
	"aziz.dev/shortener/internal/testutil"
	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestProducer creates a Producer pointed at the test broker list.
func newTestProducer(t *testing.T, brokers []string) *Producer {
	t.Helper()
	cfg := &config.KafkaConfig{Brokers: brokers}
	p, err := NewProducer(context.Background(), cfg)
	require.NoError(t, err)
	t.Cleanup(func() { _ = p.Close() })
	return p
}

// consume reads up to n messages from topic, timing out after 10 s.
func consume(t *testing.T, brokers []string, topic, groupID string, n int) []kafka.Message {
	t.Helper()
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  brokers,
		Topic:    topic,
		GroupID:  groupID,
		MinBytes: 1,
		MaxBytes: 10e6,
		MaxWait:  500 * time.Millisecond,
	})
	t.Cleanup(func() { _ = r.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var msgs []kafka.Message
	for len(msgs) < n {
		m, err := r.ReadMessage(ctx)
		require.NoError(t, err)
		msgs = append(msgs, m)
	}
	return msgs
}

// createTopic pre-creates a topic so the producer doesn't rely on auto-create.
func createTopic(t *testing.T, broker, topic string) {
	t.Helper()
	conn, err := kafka.Dial("tcp", broker)
	require.NoError(t, err)
	defer conn.Close()

	err = conn.CreateTopics(kafka.TopicConfig{
		Topic:             topic,
		NumPartitions:     1,
		ReplicationFactor: 1,
	})
	require.NoError(t, err)
}

// --- Tests ---

func TestProducer_SendEvent_DeliveredToKafka(t *testing.T) {
	ctx := context.Background()
	brokers := testutil.NewKafkaContainer(t, ctx)
	topic := "test.send-event"
	createTopic(t, brokers[0], topic)

	p := newTestProducer(t, brokers)
	payload := map[string]any{"event": "link.created", "code": "abc123"}

	err := p.SendEvent(ctx, topic, payload)
	require.NoError(t, err)

	msgs := consume(t, brokers, topic, "test-group-send", 1)
	require.Len(t, msgs, 1)

	var got map[string]any
	require.NoError(t, json.Unmarshal(msgs[0].Value, &got))
	assert.Equal(t, "link.created", got["event"])
	assert.Equal(t, "abc123", got["code"])
}

func TestProducer_SendEvent_MultipleMessages(t *testing.T) {
	ctx := context.Background()
	brokers := testutil.NewKafkaContainer(t, ctx)
	topic := "test.multi-message"
	createTopic(t, brokers[0], topic)

	p := newTestProducer(t, brokers)
	events := []map[string]any{
		{"event": "link.created", "code": "aaa"},
		{"event": "link.deleted", "code": "bbb"},
		{"event": "link.updated", "code": "ccc"},
	}

	for _, e := range events {
		require.NoError(t, p.SendEvent(ctx, topic, e))
	}

	msgs := consume(t, brokers, topic, "test-group-multi", len(events))
	assert.Len(t, msgs, len(events))
}

func TestProducer_SendEvent_PayloadIsValidJSON(t *testing.T) {
	ctx := context.Background()
	brokers := testutil.NewKafkaContainer(t, ctx)
	topic := "test.json-validity"
	createTopic(t, brokers[0], topic)

	p := newTestProducer(t, brokers)
	exp := time.Now().UTC().Truncate(time.Second)
	payload := map[string]any{
		"code":       "xyz",
		"expires_at": exp.Format(time.RFC3339),
		"is_active":  true,
		"count":      42,
	}

	require.NoError(t, p.SendEvent(ctx, topic, payload))

	msgs := consume(t, brokers, topic, "test-group-json", 1)
	var got map[string]any
	require.NoError(t, json.Unmarshal(msgs[0].Value, &got))
	assert.Equal(t, "xyz", got["code"])
	assert.Equal(t, true, got["is_active"])
	assert.Equal(t, float64(42), got["count"]) // JSON numbers decode as float64
}

func TestProducer_SendEvent_UnmarshalablePayloadReturnsError(t *testing.T) {
	ctx := context.Background()
	brokers := testutil.NewKafkaContainer(t, ctx)
	topic := "test.marshal-error"
	createTopic(t, brokers[0], topic)

	p := newTestProducer(t, brokers)

	// channels cannot be JSON-marshalled
	bad := map[string]any{"ch": make(chan int)}
	err := p.SendEvent(ctx, topic, bad)
	assert.Error(t, err)
}

func TestProducer_SendEvent_CancelledContextReturnsError(t *testing.T) {
	ctx := context.Background()
	brokers := testutil.NewKafkaContainer(t, ctx)
	topic := "test.cancelled-ctx"
	createTopic(t, brokers[0], topic)

	p := newTestProducer(t, brokers)

	cancelled, cancel := context.WithCancel(ctx)
	cancel() // cancel immediately

	err := p.SendEvent(cancelled, topic, map[string]any{"event": "x"})
	assert.Error(t, err)
}

func TestProducer_Close_IdempotentOnUnusedProducer(t *testing.T) {
	ctx := context.Background()
	brokers := testutil.NewKafkaContainer(t, ctx)
	p := newTestProducer(t, brokers)

	// First explicit close
	assert.NoError(t, p.Close())
	// t.Cleanup will call Close() again — should not panic
}