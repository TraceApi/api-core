package rest_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/TraceApi/api-core/internal/core/domain"
	"github.com/TraceApi/api-core/internal/transport/rest"
	"github.com/TraceApi/api-core/internal/transport/rest/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// --- Mocks ---

type MockPassportService struct {
	mock.Mock
}

func (m *MockPassportService) CreatePassport(ctx context.Context, manufacturerID string, category domain.ProductCategory, payload []byte) (*domain.Passport, error) {
	args := m.Called(ctx, manufacturerID, category, payload)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Passport), args.Error(1)
}

func (m *MockPassportService) GetPassport(ctx context.Context, id uuid.UUID) (*domain.Passport, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Passport), args.Error(1)
}

func (m *MockPassportService) PublishPassport(ctx context.Context, id uuid.UUID) (*domain.Passport, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Passport), args.Error(1)
}

// --- Tests ---

func TestCreatePassport_Handler_Success(t *testing.T) {
	// Setup
	mockSvc := new(MockPassportService)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	handler := rest.NewPassportHandler(mockSvc, logger)

	// Router
	r := chi.NewRouter()
	handler.RegisterRoutes(r)

	// Request Data
	payload := map[string]interface{}{"some": "data"}
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", "/passports?category=BATTERY_INDUSTRIAL", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	// Inject Auth Context (Simulate Middleware)
	ctx := context.WithValue(req.Context(), middleware.ManufacturerIDKey, "mfg-1")
	req = req.WithContext(ctx)

	// Expectations
	expectedPassport := &domain.Passport{
		ID:              uuid.New(),
		ProductCategory: domain.CategoryBattery,
		Status:          domain.StatusDraft,
	}
	mockSvc.On("CreatePassport", mock.Anything, "mfg-1", domain.CategoryBattery, mock.Anything).Return(expectedPassport, nil)

	// Execute
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	// Assertions
	assert.Equal(t, http.StatusCreated, rr.Code)

	var response domain.Passport
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, expectedPassport.ID, response.ID)
}

func TestCreatePassport_Handler_MissingCategory(t *testing.T) {
	mockSvc := new(MockPassportService)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	handler := rest.NewPassportHandler(mockSvc, logger)
	r := chi.NewRouter()
	handler.RegisterRoutes(r)

	req, _ := http.NewRequest("POST", "/passports", bytes.NewBuffer([]byte("{}"))) // No query param

	// Inject Auth Context (Even though it fails before this, it's good practice)
	ctx := context.WithValue(req.Context(), middleware.ManufacturerIDKey, "mfg-1")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "missing 'category'")
}

func TestCreatePassport_Handler_ServiceError(t *testing.T) {
	mockSvc := new(MockPassportService)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	handler := rest.NewPassportHandler(mockSvc, logger)
	r := chi.NewRouter()
	handler.RegisterRoutes(r)

	req, _ := http.NewRequest("POST", "/passports?category=BATTERY_INDUSTRIAL", bytes.NewBuffer([]byte("{}")))

	// Inject Auth Context
	ctx := context.WithValue(req.Context(), middleware.ManufacturerIDKey, "mfg-1")
	req = req.WithContext(ctx)

	mockSvc.On("CreatePassport", mock.Anything, "mfg-1", domain.CategoryBattery, mock.Anything).Return(nil, domain.ErrInvalidInput)

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestCreatePassport_Handler_InternalError(t *testing.T) {
	mockSvc := new(MockPassportService)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	handler := rest.NewPassportHandler(mockSvc, logger)
	r := chi.NewRouter()
	handler.RegisterRoutes(r)

	req, _ := http.NewRequest("POST", "/passports?category=BATTERY_INDUSTRIAL", bytes.NewBuffer([]byte("{}")))

	// Inject Auth Context
	ctx := context.WithValue(req.Context(), middleware.ManufacturerIDKey, "mfg-1")
	req = req.WithContext(ctx)

	mockSvc.On("CreatePassport", mock.Anything, "mfg-1", domain.CategoryBattery, mock.Anything).Return(nil, errors.New("db error"))

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}

func TestPublishPassport_Handler_Success(t *testing.T) {
	mockSvc := new(MockPassportService)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	handler := rest.NewPassportHandler(mockSvc, logger)
	r := chi.NewRouter()
	handler.RegisterRoutes(r)

	id := uuid.New()
	passport := &domain.Passport{ID: id, Status: domain.StatusPublished}

	req, _ := http.NewRequest("POST", "/passports/"+id.String()+"/publish", nil)

	// Inject Auth Context
	ctx := context.WithValue(req.Context(), middleware.ManufacturerIDKey, "mfg-1")
	req = req.WithContext(ctx)

	mockSvc.On("PublishPassport", mock.Anything, id).Return(passport, nil)

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	var resp domain.Passport
	json.Unmarshal(rr.Body.Bytes(), &resp)
	assert.Equal(t, id, resp.ID)
}

func TestPublishPassport_Handler_Conflict(t *testing.T) {
	mockSvc := new(MockPassportService)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	handler := rest.NewPassportHandler(mockSvc, logger)
	r := chi.NewRouter()
	handler.RegisterRoutes(r)

	id := uuid.New()

	req, _ := http.NewRequest("POST", "/passports/"+id.String()+"/publish", nil)

	// Inject Auth Context
	ctx := context.WithValue(req.Context(), middleware.ManufacturerIDKey, "mfg-1")
	req = req.WithContext(ctx)

	mockSvc.On("PublishPassport", mock.Anything, id).Return(nil, domain.ErrPassportAlreadyPublished)

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusConflict, rr.Code)
	assert.Contains(t, rr.Body.String(), "passport already published")
}
