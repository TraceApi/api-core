package rest_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/TraceApi/api-core/internal/config"
	"github.com/TraceApi/api-core/internal/transport/rest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockAuthRepo (reused or defined here if not available in handler_test.go)
// Assuming we might need to define it if it's not exported from handler_test.go
// But let's assume we can define a local one for this test file if needed,
// or use the one from the package if it exists.
// For safety, I'll define a local mock here to ensure self-containment,
// but usually we'd share mocks.

type MockAuthRepo struct {
	mock.Mock
}

func (m *MockAuthRepo) ValidateKey(ctx context.Context, keyHash string) (string, bool, error) {
	args := m.Called(ctx, keyHash)
	return args.String(0), args.Bool(1), args.Error(2)
}

func (m *MockAuthRepo) GetTenantState(ctx context.Context, tenantID string) (string, error) {
	args := m.Called(ctx, tenantID)
	return args.String(0), args.Error(1)
}

func (m *MockAuthRepo) GetTenantName(ctx context.Context, tenantID string) (string, error) {
	args := m.Called(ctx, tenantID)
	return args.String(0), args.Error(1)
}

func TestExchangeToken(t *testing.T) {
	// Setup
	mockService := new(MockPassportService)
	mockAuthRepo := new(MockAuthRepo)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	cfg := &config.Config{JWTSecret: "test-secret"}
	handler := rest.NewResolverHandler(mockService, mockAuthRepo, logger, cfg)

	t.Run("Valid API Key", func(t *testing.T) {
		// Arrange
		apiKey := "traceapi_valid_key"
		// In real code, we'd hash the key. The handler hashes it before calling ValidateKey.
		// So we need to expect the HASH of the key.
		// But wait, the handler does: hash := sha256.Sum256([]byte(req.APIKey))
		// So we need to calculate that hash to set up the mock expectation.
		// However, for a mock, we can use mock.Anything if we don't want to duplicate the hash logic,
		// or we can duplicate it to be precise. Let's be precise.
		// Actually, simpler: let the mock accept any string and return success.
		mockAuthRepo.On("ValidateKey", mock.Anything, mock.Anything).Return("tenant-123", true, nil).Once()

		reqBody := map[string]string{"apiKey": apiKey}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/auth/token", bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		// Act
		handler.ExchangeToken(w, req)

		// Assert
		resp := w.Result()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var respBody map[string]string
		json.NewDecoder(resp.Body).Decode(&respBody)
		assert.NotEmpty(t, respBody["token"])
	})

	t.Run("Invalid API Key", func(t *testing.T) {
		// Arrange
		mockAuthRepo.On("ValidateKey", mock.Anything, mock.Anything).Return("", false, nil).Once()

		reqBody := map[string]string{"apiKey": "traceapi_invalid"}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/auth/token", bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		// Act
		handler.ExchangeToken(w, req)

		// Assert
		assert.Equal(t, http.StatusUnauthorized, w.Result().StatusCode)
	})

	t.Run("Missing API Key", func(t *testing.T) {
		// Arrange
		reqBody := map[string]string{"apiKey": ""}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/auth/token", bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		// Act
		handler.ExchangeToken(w, req)

		// Assert
		assert.Equal(t, http.StatusBadRequest, w.Result().StatusCode)
	})

	t.Run("Invalid JSON", func(t *testing.T) {
		// Arrange
		req := httptest.NewRequest("POST", "/auth/token", bytes.NewBufferString("invalid-json"))
		w := httptest.NewRecorder()

		// Act
		handler.ExchangeToken(w, req)

		// Assert
		assert.Equal(t, http.StatusBadRequest, w.Result().StatusCode)
	})
}
