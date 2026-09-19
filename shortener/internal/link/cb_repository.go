package link

import (
	"context"
	"errors"
	"time"

	sqlcgen "aziz.dev/shortener/internal/postgres/sqlc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/sirupsen/logrus"
	"github.com/sony/gobreaker"
)

type CircuitBreakerRepository struct {
	underlying Repository
	cb         *gobreaker.CircuitBreaker
}

func NewCircuitBreakerRepository(underlying Repository) Repository {
	cb := gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:        "shortener-db-circuit-breaker",
		MaxRequests: 3,
		Interval:    10 * time.Second,
		Timeout:     5 * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
			return counts.Requests >= 5 && failureRatio >= 0.5
		},
		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
			logrus.Warnf("Circuit breaker %s changed state from %s to %s", name, from, to)
		},
		IsSuccessful: func(err error) bool {
			if err == nil {
				return true
			}
			
			if errors.Is(err, pgx.ErrNoRows) {
				return true
			}

			return false
		},
	})

	return &CircuitBreakerRepository{
		underlying: underlying,
		cb:         cb,
	}
}

func (r *CircuitBreakerRepository) FindAllByUserID(ctx context.Context, userID uuid.UUID, pagination Pagination) ([]sqlcgen.Link, error) {
	res, err := r.cb.Execute(func() (interface{}, error) {
		return r.underlying.FindAllByUserID(ctx, userID, pagination)
	})

	if err != nil {
		return nil, err
	}

	return res.([]sqlcgen.Link), nil
}

func (r *CircuitBreakerRepository) FindByCodeAndUserID(ctx context.Context, code string, userID uuid.UUID) (sqlcgen.Link, error) {
	res, err := r.cb.Execute(func() (interface{}, error) {
		return r.underlying.FindByCodeAndUserID(ctx, code, userID)
	})
	if err != nil {
		return sqlcgen.Link{}, err
	}

	return res.(sqlcgen.Link), nil
}

func (r *CircuitBreakerRepository) Create(ctx context.Context, arg sqlcgen.CreateLinkParams) (sqlcgen.Link, error) {
	_, err := r.cb.Execute(func() (interface{}, error) {
		return r.underlying.Create(ctx, arg)
	})

	return sqlcgen.Link{}, err
}

func (r *CircuitBreakerRepository) Update(ctx context.Context, arg sqlcgen.UpdateLinkParams) (int64, error) {
	_, err := r.cb.Execute(func() (interface{}, error) {
		return r.underlying.Update(ctx, arg)
	})

	return 0, err
}