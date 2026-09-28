package link

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// --- Mock Service ---

type MockService struct {
	mock.Mock
}

func (m *MockService) GetAll(ctx context.Context, userID uuid.UUID) ([]Dto, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]Dto), args.Error(1)
}

func (m *MockService) Create(ctx context.Context, userID uuid.UUID, req CreateRequest) (*Dto, error) {
	args := m.Called(ctx, userID, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Dto), args.Error(1)
}

func (m *MockService) UpdateExpiry(ctx context.Context, userID uuid.UUID, code string, req UpdateExpiryDto) error {
	args := m.Called(ctx, userID, code, req)
	return args.Error(0)
}

func (m *MockService) Delete(ctx context.Context, userID uuid.UUID, code string) error {
	args := m.Called(ctx, userID, code)
	return args.Error(0)
}

// --- Test Helpers ---

func setupRouter(svc Service) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewHandler(svc)
	h.RegisterRoutes(r.Group("/api"))
	return r
}

func validUserID() uuid.UUID {
	return uuid.MustParse("00000000-0000-0000-0000-000000000001")
}

// --- GetAll Tests ---

func TestGetAll_Success(t *testing.T) {
	svc := new(MockService)
	userID := validUserID()
	now := time.Now()

	expectedLinks := []Dto{
		{Code: "abc123", OriginalURL: "https://example.com", IsActive: true, CreatedAt: now},
	}
	svc.On("GetAll", mock.Anything, userID).Return(expectedLinks, nil)

	r := setupRouter(svc)
	req := httptest.NewRequest(http.MethodGet, "/api/links", nil)
	req.Header.Set("User-ID", userID.String())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var body map[string]any
	json.Unmarshal(w.Body.Bytes(), &body)
	data := body["data"].([]any)
	assert.Len(t, data, 1)
	svc.AssertExpectations(t)
}

func TestGetAll_MissingUserID(t *testing.T) {
	svc := new(MockService)
	r := setupRouter(svc)

	req := httptest.NewRequest(http.MethodGet, "/api/links", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	svc.AssertNotCalled(t, "GetAll")
}

func TestGetAll_InvalidUserID(t *testing.T) {
	svc := new(MockService)
	r := setupRouter(svc)

	req := httptest.NewRequest(http.MethodGet, "/api/links", nil)
	req.Header.Set("User-ID", "not-a-uuid")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestGetAll_ServiceError(t *testing.T) {
	svc := new(MockService)
	userID := validUserID()
	svc.On("GetAll", mock.Anything, userID).Return([]Dto{}, errors.New("db error"))

	r := setupRouter(svc)
	req := httptest.NewRequest(http.MethodGet, "/api/links", nil)
	req.Header.Set("User-ID", userID.String())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	svc.AssertExpectations(t)
}

// --- Create Tests ---

func TestCreate_Success(t *testing.T) {
	svc := new(MockService)
	userID := validUserID()
	now := time.Now()

	reqBody := CreateRequest{OriginalURL: "https://example.com"}
	expectedDto := &Dto{Code: "xyz789", OriginalURL: "https://example.com", IsActive: true, CreatedAt: now}
	svc.On("Create", mock.Anything, userID, reqBody).Return(expectedDto, nil)

	body, _ := json.Marshal(reqBody)
	r := setupRouter(svc)
	req := httptest.NewRequest(http.MethodPost, "/api/links", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-ID", userID.String())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]any)
	assert.Equal(t, "xyz789", data["code"])
	svc.AssertExpectations(t)
}

func TestCreate_MissingUserID(t *testing.T) {
	svc := new(MockService)
	r := setupRouter(svc)

	body, _ := json.Marshal(CreateRequest{OriginalURL: "https://example.com"})
	req := httptest.NewRequest(http.MethodPost, "/api/links", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	svc.AssertNotCalled(t, "Create")
}

func TestCreate_InvalidBody(t *testing.T) {
	svc := new(MockService)
	userID := validUserID()
	r := setupRouter(svc)

	// Missing required original_url
	body, _ := json.Marshal(map[string]any{"custom_alias": "alias"})
	req := httptest.NewRequest(http.MethodPost, "/api/links", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-ID", userID.String())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	svc.AssertNotCalled(t, "Create")
}

func TestCreate_InvalidURL(t *testing.T) {
	svc := new(MockService)
	userID := validUserID()
	r := setupRouter(svc)

	body, _ := json.Marshal(CreateRequest{OriginalURL: "not-a-url"})
	req := httptest.NewRequest(http.MethodPost, "/api/links", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-ID", userID.String())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreate_ServiceError(t *testing.T) {
	svc := new(MockService)
	userID := validUserID()
	reqBody := CreateRequest{OriginalURL: "https://example.com"}
	svc.On("Create", mock.Anything, userID, reqBody).Return(nil, errors.New("duplicate key"))

	body, _ := json.Marshal(reqBody)
	r := setupRouter(svc)
	req := httptest.NewRequest(http.MethodPost, "/api/links", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-ID", userID.String())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	svc.AssertExpectations(t)
}

func TestCreate_WithCustomAlias(t *testing.T) {
	svc := new(MockService)
	userID := validUserID()
	now := time.Now()

	alias := "myalias"
	reqBody := CreateRequest{OriginalURL: "https://example.com", CustomAlias: "myalias"}
	expectedDto := &Dto{Code: "myalias", OriginalURL: "https://example.com", CustomAlias: &alias, IsActive: true, CreatedAt: now}
	svc.On("Create", mock.Anything, userID, reqBody).Return(expectedDto, nil)

	body, _ := json.Marshal(reqBody)
	r := setupRouter(svc)
	req := httptest.NewRequest(http.MethodPost, "/api/links", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-ID", userID.String())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	svc.AssertExpectations(t)
}

// --- UpdateExpiry Tests ---

func TestUpdateExpiry_Success(t *testing.T) {
	svc := new(MockService)
	userID := validUserID()
	code := "abc123"
	exp := time.Now().Add(24 * time.Hour)
	reqDto := UpdateExpiryDto{ExpiresAt: &exp}

	// Use MatchedBy to compare the UpdateExpiryDto without monotonic clock
	svc.On("UpdateExpiry", mock.Anything, userID, code, mock.MatchedBy(func(dto UpdateExpiryDto) bool {
		if dto.ExpiresAt == nil && reqDto.ExpiresAt == nil {
			return true
		}
		if dto.ExpiresAt == nil || reqDto.ExpiresAt == nil {
			return false
		}
		// Compare times without monotonic clock
		return dto.ExpiresAt.Truncate(time.Second).Equal(reqDto.ExpiresAt.Truncate(time.Second))
	})).Return(nil)

	body, _ := json.Marshal(reqDto)
	r := setupRouter(svc)
	req := httptest.NewRequest(http.MethodPatch, "/api/links/"+code+"/expiry", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-ID", userID.String())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "successfully updated expiry", resp["message"])
	svc.AssertExpectations(t)
}

func TestUpdateExpiry_MissingUserID(t *testing.T) {
	svc := new(MockService)
	r := setupRouter(svc)
	exp := time.Now().Add(24 * time.Hour)
	body, _ := json.Marshal(UpdateExpiryDto{ExpiresAt: &exp})

	req := httptest.NewRequest(http.MethodPatch, "/api/links/abc123/expiry", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	svc.AssertNotCalled(t, "UpdateExpiry")
}

func TestUpdateExpiry_InvalidBody(t *testing.T) {
	svc := new(MockService)
	userID := validUserID()
	r := setupRouter(svc)

	// Missing required expires_at
	body, _ := json.Marshal(map[string]any{})
	req := httptest.NewRequest(http.MethodPatch, "/api/links/abc123/expiry", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-ID", userID.String())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateExpiry_ServiceError(t *testing.T) {
	svc := new(MockService)
	userID := validUserID()
	code := "abc123"
	exp := time.Now().Add(24 * time.Hour)
	reqDto := UpdateExpiryDto{ExpiresAt: &exp}
	// Use MatchedBy to compare the UpdateExpiryDto without monotonic clock
	svc.On("UpdateExpiry", mock.Anything, userID, code, mock.MatchedBy(func(dto UpdateExpiryDto) bool {
		if dto.ExpiresAt == nil && reqDto.ExpiresAt == nil {
			return true
		}
		if dto.ExpiresAt == nil || reqDto.ExpiresAt == nil {
			return false
		}
		// Compare times without monotonic clock
		return dto.ExpiresAt.Truncate(time.Second).Equal(reqDto.ExpiresAt.Truncate(time.Second))
	})).Return(errors.New("link not found or unauthorized"))

	body, _ := json.Marshal(reqDto)
	r := setupRouter(svc)
	req := httptest.NewRequest(http.MethodPatch, "/api/links/"+code+"/expiry", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-ID", userID.String())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	svc.AssertExpectations(t)
}

// --- Delete Tests ---

func TestDelete_Success(t *testing.T) {
	svc := new(MockService)
	userID := validUserID()
	code := "abc123"
	svc.On("Delete", mock.Anything, userID, code).Return(nil)

	r := setupRouter(svc)
	req := httptest.NewRequest(http.MethodDelete, "/api/links/"+code, nil)
	req.Header.Set("User-ID", userID.String())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "successfully deleted link", resp["message"])
	svc.AssertExpectations(t)
}

func TestDelete_MissingUserID(t *testing.T) {
	svc := new(MockService)
	r := setupRouter(svc)

	req := httptest.NewRequest(http.MethodDelete, "/api/links/abc123", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	svc.AssertNotCalled(t, "Delete")
}

func TestDelete_ServiceError(t *testing.T) {
	svc := new(MockService)
	userID := validUserID()
	code := "abc123"
	svc.On("Delete", mock.Anything, userID, code).Return(errors.New("link not found or unauthorized"))

	r := setupRouter(svc)
	req := httptest.NewRequest(http.MethodDelete, "/api/links/"+code, nil)
	req.Header.Set("User-ID", userID.String())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	svc.AssertExpectations(t)
}
