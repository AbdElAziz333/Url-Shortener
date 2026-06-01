package testutil

import (
	"context"
	"testing"

	mongoApi "go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/mongodb"
)

func NewMongoContainer(t *testing.T, ctx context.Context) *mongoApi.Client {
	t.Helper()

	mongoContainer, err := mongodb.Run(ctx, "mongo:7.0")
	require.NoError(t, err)

	endpoint, err := mongoContainer.Endpoint(ctx, "")
	require.NoError(t, err)

	client, err := mongoApi.Connect(options.Client().ApplyURI(endpoint))
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = mongoContainer.Terminate(ctx)
	})

	// t.Cleanup(func() {
	// 	if err := client.Disconnect(ctx); err != nil {
	// 		t.Errorf("error disconnecting from MongoDB: %v", err)
	// 	}
	// 	if err := mongoContainer.Terminate(ctx); err != nil {
	// 		t.Errorf("error terminating MongoDB container: %v", err)
	// 	}
	// })

	return client
}