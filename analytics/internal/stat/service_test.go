package stat

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockRepository struct {
	mock.Mock
}

func (m *mockRepository) GetTotalClicks(ctx context.Context, code string) ([]Dto, error) {
	args := m.Called(ctx, code)
	return args.Get(0).([]Dto), args.Error(1)
}

func (m *mockRepository) GetGeo(ctx context.Context, code string) ([]Dto, error) {
	args := m.Called(ctx, code)
	return args.Get(0).([]Dto), args.Error(1)
}

func (m *mockRepository) GetReferrers(ctx context.Context, code string) ([]Dto, error) {
	args := m.Called(ctx, code)
	return args.Get(0).([]Dto), args.Error(1)
}

func (m *mockRepository) SaveClickEvent(ctx context.Context, event map[string]any) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

// ---------------------------------------------------------------------------
// GetTotalClicks
// ---------------------------------------------------------------------------

func TestService_GetTotalClicks_Success(t *testing.T) {
	repo := new(mockRepository)
	ctx := context.Background()
	expected := []Dto{{Key: "total", Count: 99}}
	repo.On("GetTotalClicks", ctx, "code1").Return(expected, nil)

	svc := NewService(repo)
	result, err := svc.GetTotalClicks(ctx, "code1")

	assert.NoError(t, err)
	assert.Equal(t, expected, result)
	repo.AssertExpectations(t)
}

func TestService_GetTotalClicks_RepoError(t *testing.T) {
	repo := new(mockRepository)
	ctx := context.Background()
	repo.On("GetTotalClicks", ctx, "code1").Return([]Dto{}, errors.New("db error"))

	svc := NewService(repo)
	result, err := svc.GetTotalClicks(ctx, "code1")

	assert.Error(t, err)
	assert.Equal(t, "db error", err.Error())
	assert.Empty(t, result)
	repo.AssertExpectations(t)
}

func TestService_GetTotalClicks_Zero(t *testing.T) {
	repo := new(mockRepository)
	ctx := context.Background()
	repo.On("GetTotalClicks", ctx, "new").Return([]Dto{{Key: "total", Count: 0}}, nil)

	svc := NewService(repo)
	result, err := svc.GetTotalClicks(ctx, "new")

	assert.NoError(t, err)
	assert.Equal(t, int64(0), result[0].Count)
	repo.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// GetGeo
// ---------------------------------------------------------------------------

func TestService_GetGeo_Success(t *testing.T) {
	repo := new(mockRepository)
	ctx := context.Background()
	expected := []Dto{
		{Key: "US", Count: 300},
		{Key: "GB", Count: 120},
	}
	repo.On("GetGeo", ctx, "code1").Return(expected, nil)

	svc := NewService(repo)
	result, err := svc.GetGeo(ctx, "code1")

	assert.NoError(t, err)
	assert.Equal(t, expected, result)
	repo.AssertExpectations(t)
}

func TestService_GetGeo_RepoError(t *testing.T) {
	repo := new(mockRepository)
	ctx := context.Background()
	repo.On("GetGeo", ctx, "code1").Return([]Dto{}, errors.New("mongo down"))

	svc := NewService(repo)
	result, err := svc.GetGeo(ctx, "code1")

	assert.Error(t, err)
	assert.Empty(t, result)
	repo.AssertExpectations(t)
}

func TestService_GetGeo_UnknownCountry(t *testing.T) {
	repo := new(mockRepository)
	ctx := context.Background()
	expected := []Dto{{Key: "unknown", Count: 7}}
	repo.On("GetGeo", ctx, "code1").Return(expected, nil)

	svc := NewService(repo)
	result, err := svc.GetGeo(ctx, "code1")

	assert.NoError(t, err)
	assert.Equal(t, "unknown", result[0].Key)
	repo.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// GetReferrers
// ---------------------------------------------------------------------------

func TestService_GetReferrers_Success(t *testing.T) {
	repo := new(mockRepository)
	ctx := context.Background()
	expected := []Dto{
		{Key: "google.com", Count: 500},
		{Key: "reddit.com", Count: 200},
	}
	repo.On("GetReferrers", ctx, "code1").Return(expected, nil)

	svc := NewService(repo)
	result, err := svc.GetReferrers(ctx, "code1")

	assert.NoError(t, err)
	assert.Equal(t, expected, result)
	repo.AssertExpectations(t)
}

func TestService_GetReferrers_RepoError(t *testing.T) {
	repo := new(mockRepository)
	ctx := context.Background()
	repo.On("GetReferrers", ctx, "code1").Return([]Dto{}, errors.New("connection refused"))

	svc := NewService(repo)
	result, err := svc.GetReferrers(ctx, "code1")

	assert.Error(t, err)
	assert.Empty(t, result)
	repo.AssertExpectations(t)
}

func TestService_GetReferrers_Empty(t *testing.T) {
	repo := new(mockRepository)
	ctx := context.Background()
	repo.On("GetReferrers", ctx, "fresh").Return([]Dto{}, nil)

	svc := NewService(repo)
	result, err := svc.GetReferrers(ctx, "fresh")

	assert.NoError(t, err)
	assert.Empty(t, result)
	repo.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// Context propagation
// ---------------------------------------------------------------------------

func TestService_ContextPropagation(t *testing.T) {
	// Ensures the service passes the exact context through to the repo.
	repo := new(mockRepository)
	type ctxKey struct{}
	ctx := context.WithValue(context.Background(), ctxKey{}, "sentinel")

	repo.On("GetTotalClicks", mock.MatchedBy(func(c context.Context) bool {
		return c.Value(ctxKey{}) == "sentinel"
	}), "code").Return([]Dto{{Key: "total", Count: 1}}, nil)

	svc := NewService(repo)
	_, err := svc.GetTotalClicks(ctx, "code")

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}