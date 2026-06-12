package link

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/sony/gobreaker"
	"gorm.io/gorm"
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
			// gorm.ErrRecordNotFound is an expected business/lookup result, not a DB connection failure.
			if errors.Is(err, gorm.ErrRecordNotFound) {
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

func (r *CircuitBreakerRepository) FindAllByUserID(ctx context.Context, userID uuid.UUID, pagination Pagination) (*[]Link, error) {
	res, err := r.cb.Execute(func() (interface{}, error) {
		return r.underlying.FindAllByUserID(ctx, userID, pagination)
	})
	if err != nil {
		return nil, err
	}
	return res.(*[]Link), nil
}

func (r *CircuitBreakerRepository) FindByCodeAndUserID(ctx context.Context, code string, userID uuid.UUID) (*Link, error) {
	res, err := r.cb.Execute(func() (interface{}, error) {
		return r.underlying.FindByCodeAndUserID(ctx, code, userID)
	})
	if err != nil {
		return nil, err
	}
	return res.(*Link), nil
}

func (r *CircuitBreakerRepository) Create(ctx context.Context, link *Link) error {
	_, err := r.cb.Execute(func() (interface{}, error) {
		return nil, r.underlying.Create(ctx, link)
	})
	return err
}

func (r *CircuitBreakerRepository) Update(ctx context.Context, link *Link) error {
	_, err := r.cb.Execute(func() (interface{}, error) {
		return nil, r.underlying.Update(ctx, link)
	})
	return err
}
