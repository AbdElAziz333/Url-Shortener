package resolve

import (
	"context"
	"errors"
	"time"

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
		Name:        "redirect-db-circuit-breaker",
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
			// gorm.ErrRecordNotFound and ErrNotFound are expected business/lookup results, not DB connection failures.
			if errors.Is(err, gorm.ErrRecordNotFound) || errors.Is(err, ErrNotFound) {
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

func (r *CircuitBreakerRepository) Find(ctx context.Context, code string) (*Link, error) {
	res, err := r.cb.Execute(func() (interface{}, error) {
		return r.underlying.Find(ctx, code)
	})
	if err != nil {
		return nil, err
	}
	return res.(*Link), nil
}
