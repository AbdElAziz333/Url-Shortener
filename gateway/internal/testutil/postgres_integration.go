package testutil

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func NewPostgres(t *testing.T, ctx context.Context) *pgxpool.Pool {
    t.Helper()

    postgresContainer, err := postgres.Run(ctx, "postgres:16-alpine",
        postgres.WithDatabase("gateway_test"),
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

    db, err := pgxpool.New(ctx, dsn)
    require.NoError(t, err)

    return db
}