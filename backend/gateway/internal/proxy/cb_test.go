package proxy

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

type mockRoundTripper struct {
	roundTripFunc func(req *http.Request) (*http.Response, error)
}

func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.roundTripFunc(req)
}

func TestCircuitBreakerRoundTripper_ClosedStateSuccess(t *testing.T) {
	mock := &mockRoundTripper{
		roundTripFunc: func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader([]byte("success"))),
			}, nil
		},
	}

	rt := NewCircuitBreakerRoundTripper("test-cb-success", mock)
	req, _ := http.NewRequest(http.MethodGet, "http://example.com", nil)
	resp, err := rt.RoundTrip(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	assert.Equal(t, "success", string(body))
}

func TestCircuitBreakerRoundTripper_TripsAndFailsFast(t *testing.T) {
	var count int
	mock := &mockRoundTripper{
		roundTripFunc: func(req *http.Request) (*http.Response, error) {
			count++
			return nil, errors.New("network error")
		},
	}

	rt := NewCircuitBreakerRoundTripper("test-cb-trips", mock)
	req, _ := http.NewRequest(http.MethodGet, "http://example.com", nil)

	// Send 5 failing requests to trip the breaker (configured as requests >= 5 and failure ratio >= 50%)
	for i := 0; i < 5; i++ {
		_, err := rt.RoundTrip(req)
		assert.Error(t, err)
	}

	// The 6th request should fail fast with a 503 response because the circuit breaker is now Open
	resp, err := rt.RoundTrip(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(body), "service temporarily unavailable due to open circuit breaker")

	// Verify the underlying transport was NOT called for the 6th request
	assert.Equal(t, 5, count)
}

func TestCircuitBreakerRoundTripper_AppFailureTrips(t *testing.T) {
	mock := &mockRoundTripper{
		roundTripFunc: func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusInternalServerError,
				Body:       io.NopCloser(bytes.NewReader([]byte("error"))),
			}, nil
		},
	}

	rt := NewCircuitBreakerRoundTripper("test-cb-app", mock)
	req, _ := http.NewRequest(http.MethodGet, "http://example.com", nil)

	// Send 5 failing requests (500 Internal Server Error)
	for i := 0; i < 5; i++ {
		resp, err := rt.RoundTrip(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	}

	// 6th request should fail fast with 503
	resp, err := rt.RoundTrip(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
}
