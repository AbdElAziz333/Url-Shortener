package resolve

import (
	"context"
	"errors"
	"testing"

	"github.com/sony/gobreaker"
	"github.com/stretchr/testify/assert"
)

type mockResolveRepository struct {
	findFunc func(ctx context.Context, code string) (*Link, error)
}

func (m *mockResolveRepository) Find(ctx context.Context, code string) (*Link, error) {
	return m.findFunc(ctx, code)
}

func TestCircuitBreakerRepository_Success(t *testing.T) {
	mock := &mockResolveRepository{
		findFunc: func(ctx context.Context, code string) (*Link, error) {
			return &Link{Code: "test-code"}, nil
		},
	}

	cbRepo := NewCircuitBreakerRepository(mock)
	link, err := cbRepo.Find(context.Background(), "test-code")

	assert.NoError(t, err)
	assert.Equal(t, "test-code", link.Code)
}

func TestCircuitBreakerRepository_TripsAndFailsFast(t *testing.T) {
	mock := &mockResolveRepository{
		findFunc: func(ctx context.Context, code string) (*Link, error) {
			return nil, errors.New("db crash")
		},
	}

	cbRepo := NewCircuitBreakerRepository(mock)

	// Send 5 failing requests to trip the breaker (configured as requests >= 5 and failure ratio >= 50%)
	for i := 0; i < 5; i++ {
		_, err := cbRepo.Find(context.Background(), "test-code")
		assert.Error(t, err)
		assert.Equal(t, "db crash", err.Error())
	}

	// 6th request should fail fast with ErrOpenState
	_, err := cbRepo.Find(context.Background(), "test-code")
	assert.Error(t, err)
	assert.True(t, errors.Is(err, gobreaker.ErrOpenState))
}

func TestCircuitBreakerRepository_DoesNotTripOnRecordNotFound(t *testing.T) {
	mock := &mockResolveRepository{
		findFunc: func(ctx context.Context, code string) (*Link, error) {
			return nil, ErrNotFound
		},
	}

	cbRepo := NewCircuitBreakerRepository(mock)

	// Send 5 ErrNotFound errors. They should NOT trip the breaker.
	for i := 0; i < 5; i++ {
		_, err := cbRepo.Find(context.Background(), "test-code")
		assert.True(t, errors.Is(err, ErrNotFound))
	}

	// 6th request should still call the mock (and return ErrNotFound) rather than ErrOpenState
	_, err := cbRepo.Find(context.Background(), "test-code")
	assert.True(t, errors.Is(err, ErrNotFound))
}
