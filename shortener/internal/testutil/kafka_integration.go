package testutil

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/kafka"
)

func NewKafkaContainer(t *testing.T, ctx context.Context) []string {
	t.Helper()

	kafkaContainer, err := kafka.Run(ctx, "confluentinc/confluent-local:7.5.0", kafka.WithClusterID("test-cluster"))
	require.NoError(t, err)

	brokers, err := kafkaContainer.Brokers(ctx)
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = kafkaContainer.Terminate(ctx)
	})

	return brokers
}
