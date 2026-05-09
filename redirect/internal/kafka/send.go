package kafka

import (
	"context"
	"encoding/json"

	"github.com/segmentio/kafka-go"
)

func (p *Producer) SendEvent(ctx context.Context, topic string, message map[string]any) error {
	b, err := json.Marshal(message)
	if err != nil {
		return err
	}

	return p.writer.WriteMessages(ctx, kafka.Message{
		Topic: topic,
		Value: b,
	})

	return nil
}