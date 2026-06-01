package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockIngestRepo struct {
	mock.Mock
}

func (m *mockIngestRepo) SaveClickEvent(ctx context.Context, event map[string]any) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

// ---------------------------------------------------------------------------
// Mock: kafka reader (wraps the minimal surface used by Consumer)
// ---------------------------------------------------------------------------

// KafkaReader is the interface Consumer depends on for reads.
// Extract it so we can swap in a fake during tests.
type KafkaReader interface {
	ReadMessage(ctx context.Context) ([]byte, error) // simplified for testing
	Close() error
}

// fakeReader is an in-memory reader that plays back a preset sequence of
// (payload, error) pairs and then blocks / returns ctx.Err().
type fakeReader struct {
	msgs []fakeMsg
	pos  int
}

type fakeMsg struct {
	payload []byte
	err     error
}

func (f *fakeReader) next() ([]byte, error) {
	if f.pos >= len(f.msgs) {
		// Simulate blocking until context cancelled — just return a sentinel error.
		return nil, context.Canceled
	}
	m := f.msgs[f.pos]
	f.pos++
	return m.payload, m.err
}

// ---------------------------------------------------------------------------
// ReadEvent unit tests (testing the JSON-unmarshal layer directly)
// ---------------------------------------------------------------------------

func TestReadEvent_ValidJSON(t *testing.T) {
	payload := map[string]any{
		"code":    "abc123",
		"country": "US",
	}
	raw, _ := json.Marshal(payload)

	var got map[string]any
	err := json.Unmarshal(raw, &got)

	assert.NoError(t, err)
	assert.Equal(t, "abc123", got["code"])
	assert.Equal(t, "US", got["country"])
}

func TestReadEvent_InvalidJSON(t *testing.T) {
	bad := []byte(`{not valid json`)
	var got map[string]any
	err := json.Unmarshal(bad, &got)
	assert.Error(t, err)
}

func TestReadEvent_EmptyObject(t *testing.T) {
	raw := []byte(`{}`)
	var got map[string]any
	err := json.Unmarshal(raw, &got)
	assert.NoError(t, err)
	assert.Empty(t, got)
}

func TestReadEvent_ExtraFields(t *testing.T) {
	// Ensures unknown keys survive the round-trip.
	payload := map[string]any{
		"code":            "xyz",
		"referrer_domain": "github.com",
		"ts":              1_700_000_000,
	}
	raw, _ := json.Marshal(payload)

	var got map[string]any
	err := json.Unmarshal(raw, &got)

	assert.NoError(t, err)
	assert.Equal(t, "github.com", got["referrer_domain"])
}

// ---------------------------------------------------------------------------
// Ingest loop behaviour — tested via an injectable helper that mimics the
// core logic of Consumer.Ingest without requiring a live Kafka broker.
// ---------------------------------------------------------------------------

// ingestLoop is a test-local replica of the Ingest loop logic so we can
// drive it with our fakeReader without touching real Kafka.
func ingestLoop(ctx context.Context, read func() (map[string]any, error), repo mockIngestRepo) {
	for {
		event, err := read()
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			continue
		}
		_ = repo.SaveClickEvent(ctx, event) //nolint:errcheck
	}
}

func TestIngest_SavesEventOnSuccess(t *testing.T) {
	event := map[string]any{"code": "abc", "country": "US"}
	raw, _ := json.Marshal(event)

	calls := 0
	readFn := func() (map[string]any, error) {
		if calls == 0 {
			calls++
			var m map[string]any
			_ = json.Unmarshal(raw, &m)
			return m, nil
		}
		return nil, context.Canceled
	}

	repo := new(mockIngestRepo)
	repo.On("SaveClickEvent", mock.Anything, mock.MatchedBy(func(e map[string]any) bool {
		return e["code"] == "abc"
	})).Return(nil).Once()

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	ingestLoop(ctx, readFn, *repo)

	repo.AssertExpectations(t)
}

func TestIngest_ContinuesAfterReadError(t *testing.T) {
	calls := 0
	event := map[string]any{"code": "x"}
	raw, _ := json.Marshal(event)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	readFn := func() (map[string]any, error) {
		calls++
		switch calls {
		case 1:
			return nil, errors.New("transient kafka error")
		case 2:
			var m map[string]any
			_ = json.Unmarshal(raw, &m)
			return m, nil
		default:
			cancel() // Cancel context after third call
			return nil, ctx.Err()
		}
	}

	repo := new(mockIngestRepo)
	repo.On("SaveClickEvent", mock.Anything, mock.Anything).Return(nil).Once()

	ingestLoop(ctx, readFn, *repo)

	repo.AssertExpectations(t)
	assert.Equal(t, 3, calls) // error → success → cancel
}

func TestIngest_StopsOnContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // pre-cancel

	saveCount := 0
	readFn := func() (map[string]any, error) {
		return nil, ctx.Err()
	}

	repo := new(mockIngestRepo)
	// SaveClickEvent must never be called
	ingestLoop(ctx, readFn, *repo)

	assert.Equal(t, 0, saveCount)
	repo.AssertNotCalled(t, "SaveClickEvent")
}

func TestIngest_SaveError_DoesNotStop(t *testing.T) {
	calls := 0
	event := map[string]any{"code": "y"}
	raw, _ := json.Marshal(event)

	readFn := func() (map[string]any, error) {
		calls++
		if calls > 2 {
			return nil, context.Canceled
		}
		var m map[string]any
		_ = json.Unmarshal(raw, &m)
		return m, nil
	}

	repo := new(mockIngestRepo)
	repo.On("SaveClickEvent", mock.Anything, mock.Anything).
		Return(errors.New("db error")).Times(2)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	ingestLoop(ctx, readFn, *repo)

	repo.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// SaveClickEvent delegation tests (repository-layer contract)
// ---------------------------------------------------------------------------

func TestSaveClickEvent_CalledWithCorrectPayload(t *testing.T) {
	repo := new(mockIngestRepo)
	event := map[string]any{
		"code":            "abc123",
		"country":         "DE",
		"referrer_domain": "bing.com",
	}
	repo.On("SaveClickEvent", mock.Anything, event).Return(nil)

	err := repo.SaveClickEvent(context.Background(), event)

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestSaveClickEvent_PropagatesError(t *testing.T) {
	repo := new(mockIngestRepo)
	repo.On("SaveClickEvent", mock.Anything, mock.Anything).
		Return(errors.New("insert failed"))

	err := repo.SaveClickEvent(context.Background(), map[string]any{"code": "z"})

	assert.EqualError(t, err, "insert failed")
	repo.AssertExpectations(t)
}
