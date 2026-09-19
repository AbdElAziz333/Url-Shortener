package link

import (
	"context"
	"errors"
	"testing"

	sqlcgen "aziz.dev/shortener/internal/postgres/sqlc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/sony/gobreaker"
	"github.com/stretchr/testify/assert"
)

type mockRepository struct {
	findAllByUserIDFunc     func(ctx context.Context, userID uuid.UUID, pagination Pagination) ([]sqlcgen.Link, error)
	findByCodeAndUserIDFunc func(ctx context.Context, code string, userID uuid.UUID) (sqlcgen.Link, error)
	createFunc              func(ctx context.Context, link *sqlcgen.Link) (sqlcgen.Link, error)
	updateFunc              func(ctx context.Context, link *sqlcgen.Link) (int64, error)
}

func (m *mockRepository) FindAllByUserID(ctx context.Context, userID uuid.UUID, pagination Pagination) ([]sqlcgen.Link, error) {
	return m.findAllByUserIDFunc(ctx, userID, pagination)
}

func (m *mockRepository) FindByCodeAndUserID(ctx context.Context, code string, userID uuid.UUID) (sqlcgen.Link, error) {
	return m.findByCodeAndUserIDFunc(ctx, code, userID)
}

func (m *mockRepository) Create(ctx context.Context, link *sqlcgen.Link) (sqlcgen.Link, error) {
	return m.createFunc(ctx, link)
}

func (m *mockRepository) Update(ctx context.Context, link *sqlcgen.Link) (int64, error) {
	return m.updateFunc(ctx, link)
}

func TestCircuitBreakerRepository_Success(t *testing.T) {
	mock := &mockRepository{
		findByCodeAndUserIDFunc: func(ctx context.Context, code string, userID uuid.UUID) (sqlcgen.Link, error) {
			return sqlcgen.Link{Code: "test-code"}, nil
		},
	}

	cbRepo := NewCircuitBreakerRepository(mock)
	link, err := cbRepo.FindByCodeAndUserID(context.Background(), "test-code", uuid.New())

	assert.NoError(t, err)
	assert.Equal(t, "test-code", link.Code)
}

func TestCircuitBreakerRepository_TripsAndFailsFast(t *testing.T) {
	mock := &mockRepository{
		findByCodeAndUserIDFunc: func(ctx context.Context, code string, userID uuid.UUID) (sqlcgen.Link, error) {
			return sqlcgen.Link{}, errors.New("db crash")
		},
	}

	cbRepo := NewCircuitBreakerRepository(mock)

	// Send 5 failing requests to trip the breaker (configured as requests >= 5 and failure ratio >= 50%)
	for i := 0; i < 5; i++ {
		_, err := cbRepo.FindByCodeAndUserID(context.Background(), "test-code", uuid.New())
		assert.Error(t, err)
		assert.Equal(t, "db crash", err.Error())
	}

	// 6th request should fail fast with ErrOpenState
	_, err := cbRepo.FindByCodeAndUserID(context.Background(), "test-code", uuid.New())
	assert.Error(t, err)
	assert.True(t, errors.Is(err, gobreaker.ErrOpenState))
}

func TestCircuitBreakerRepository_DoesNotTripOnRecordNotFound(t *testing.T) {
	mock := &mockRepository{
		findByCodeAndUserIDFunc: func(ctx context.Context, code string, userID uuid.UUID) (sqlcgen.Link, error) {
			return sqlcgen.Link{}, pgx.ErrNoRows
		},
	}

	cbRepo := NewCircuitBreakerRepository(mock)

	for i := 0; i < 5; i++ {
		_, err := cbRepo.FindByCodeAndUserID(context.Background(), "test-code", uuid.New())
		assert.True(t, errors.Is(err, pgx.ErrNoRows))
	}

	// 6th request should still call the mock (and return pgx.ErrNoRows) rather than ErrOpenState
	_, err := cbRepo.FindByCodeAndUserID(context.Background(), "test-code", uuid.New())
	assert.True(t, errors.Is(err, pgx.ErrNoRows))
}