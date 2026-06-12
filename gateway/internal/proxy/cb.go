package proxy

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/sony/gobreaker"
)

// HTTPError wraps an HTTP response and an error so that the circuit breaker
// counts it as a failure while allowing us to recover and return the original
// response to the caller.
type HTTPError struct {
	Response *http.Response
	Err      error
}

func (e *HTTPError) Error() string {
	return e.Err.Error()
}

type CircuitBreakerRoundTripper struct {
	cb        *gobreaker.CircuitBreaker
	transport http.RoundTripper
}

func NewCircuitBreakerRoundTripper(name string, transport http.RoundTripper) *CircuitBreakerRoundTripper {
	if transport == nil {
		transport = http.DefaultTransport
	}

	cb := gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:        name,
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
	})

	return &CircuitBreakerRoundTripper{
		cb:        cb,
		transport: transport,
	}
}

func (rt *CircuitBreakerRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	res, err := rt.cb.Execute(func() (interface{}, error) {
		resp, err := rt.transport.RoundTrip(req)
		if err != nil {
			return nil, err
		}

		// Consider 5xx Server Errors as failures for the circuit breaker
		if resp.StatusCode >= 500 {
			return nil, &HTTPError{
				Response: resp,
				Err:      fmt.Errorf("HTTP status %d", resp.StatusCode),
			}
		}

		return resp, nil
	})

	if err != nil {
		// If it's the circuit breaker's own open state error, return 503 fallback
		if errors.Is(err, gobreaker.ErrOpenState) {
			logrus.Errorf("Circuit breaker %s is open, blocking request to %s", rt.cb.Name(), req.URL.String())

			bodyBytes := []byte(`{"error":"service temporarily unavailable due to open circuit breaker"}`)
			resp := &http.Response{
				StatusCode:    http.StatusServiceUnavailable,
				Status:        "503 Service Unavailable",
				Proto:         req.Proto,
				ProtoMajor:    req.ProtoMajor,
				ProtoMinor:    req.ProtoMinor,
				Body:          io.NopCloser(bytes.NewReader(bodyBytes)),
				ContentLength: int64(len(bodyBytes)),
				Header:        make(http.Header),
				Request:       req,
			}
			resp.Header.Set("Content-Type", "application/json")
			return resp, nil
		}

		// If it is a wrapped HTTPError, return the original response to the client
		// while the circuit breaker has recorded it as a failure.
		var httpErr *HTTPError
		if errors.As(err, &httpErr) {
			return httpErr.Response, nil
		}

		// Return any other transport/network errors
		return nil, err
	}

	return res.(*http.Response), nil
}
