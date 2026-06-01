package testutil

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/kafka"
)

func NewKafkaContainer(t *testing.T, ctx context.Context) []string {
	t.Helper()

	kafkaContainer, err := kafka.Run(ctx, "apache/kafka:3.9.0")
	require.NoError(t, err)

	brokers, err := kafkaContainer.Brokers(ctx)
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = kafkaContainer.Terminate(ctx)
	})

	return brokers
}