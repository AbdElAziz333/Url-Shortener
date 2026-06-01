package testutil

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	gormpostgres "gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func NewPostgres(t *testing.T, ctx context.Context) *gorm.DB {
    t.Helper()

    postgresContainer, err := postgres.Run(ctx, "postgres:16-alpine",
        postgres.WithDatabase("shortener_test"),
        postgres.WithUsername("test"),
        postgres.WithPassword("test"),
        testcontainers.WithWaitStrategy(
            wait.ForLog("database system is ready to accept connections").
                WithOccurrence(2).
                WithStartupTimeout(90*time.Second),
        ),
    )
    require.NoError(t, err)

    t.Cleanup(func() { 
        _ = postgresContainer.Terminate(ctx)
    })

    dsn, err := postgresContainer.ConnectionString(ctx, "sslmode=disable")
    require.NoError(t, err)

    db, err := gorm.Open(gormpostgres.Open(dsn), &gorm.Config{})
    require.NoError(t, err)

    return db
}