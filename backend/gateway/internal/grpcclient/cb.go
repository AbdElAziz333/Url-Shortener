package grpcclient

import (
	"context"
	"errors"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/sony/gobreaker"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func isServerError(err error) bool {
	st, ok := status.FromError(err)
	if !ok {
		return true
	}

	switch st.Code() {
	case codes.Internal, codes.Unavailable, codes.DeadlineExceeded, codes.ResourceExhausted, codes.Aborted, codes.DataLoss:
		return true
	default:
		return false
	}
}

func circuitBreakerUnaryInterceptor(name string) grpc.UnaryClientInterceptor {
	cb := gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:        name,
		MaxRequests: 3,
		Interval:    10 * time.Second,
		Timeout:     5 * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
			return counts.Requests >= 5 && failureRatio >= 0.5
		},
		OnStateChange: func(cbName string, from gobreaker.State, to gobreaker.State) {
			logrus.Warnf("Circuit breaker %s changed state from %s to %s", cbName, from, to)
		},
	})

	return func(
		ctx context.Context,
		method string,
		req, reply any,
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		result, err := cb.Execute(func() (any, error) {
			err := invoker(ctx, method, req, reply, cc, opts...)
			if err == nil {
				return nil, nil
			}
			if isServerError(err) {
				return nil, err
			}
			// Client errors should not trip the circuit breaker.
			return err, nil
		})

		if err == nil {
			if callErr, ok := result.(error); ok {
				return callErr
			}
			return nil
		}

		if errors.Is(err, gobreaker.ErrOpenState) {
			logrus.Errorf("Circuit breaker %s is open, blocking gRPC call %s", name, method)
			return status.Error(codes.Unavailable, "service temporarily unavailable due to open circuit breaker")
		}

		return err
	}
}
