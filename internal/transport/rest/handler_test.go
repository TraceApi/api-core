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
	req.Header.Set("X-Manufacturer-ID", "mfg-1")

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

	// Simulate Domain Error (Invalid Input)
	mockSvc.On("CreatePassport", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, domain.ErrInvalidInput)

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "invalid input")
}

func TestCreatePassport_Handler_InternalError(t *testing.T) {
	mockSvc := new(MockPassportService)
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	handler := rest.NewPassportHandler(mockSvc, logger)
	r := chi.NewRouter()
	handler.RegisterRoutes(r)

	req, _ := http.NewRequest("POST", "/passports?category=BATTERY_INDUSTRIAL", bytes.NewBuffer([]byte("{}")))

	// Simulate Unexpected Error
	mockSvc.On("CreatePassport", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("db exploded"))

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	assert.Contains(t, rr.Body.String(), "internal server error")
}
