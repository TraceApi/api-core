/*
 * Copyright (c) 2025 Alessandro Faranda Gancio (dba TraceApi)
 *
 * This source code is licensed under the Business Source License 1.1.
 *
 * Change Date: 2027-11-28
 * Change License: AGPL-3.0
 */

package middleware

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockAuthRepository
type MockAuthRepository struct {
	mock.Mock
}

func (m *MockAuthRepository) ValidateKey(ctx context.Context, apiKeyHash string) (string, bool, error) {
	args := m.Called(ctx, apiKeyHash)
	return args.String(0), args.Bool(1), args.Error(2)
}

func TestHybridAuthMiddleware(t *testing.T) {
	secret := "test-secret"
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	mockRepo := new(MockAuthRepository)
	middleware := HybridAuthMiddleware(secret, mockRepo, logger)

	// Helper to create a token
	createToken := func(secret string, sub string, exp time.Duration) string {
		claims := jwt.MapClaims{
			"sub": sub,
			"exp": time.Now().Add(exp).Unix(),
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, _ := token.SignedString([]byte(secret))
		return tokenString
	}

	// Helper to create API Key Hash
	createKeyHash := func(key string) string {
		hash := sha256.Sum256([]byte(key))
		return hex.EncodeToString(hash[:])
	}

	tests := []struct {
		name           string
		authHeader     string
		setupMock      func()
		expectedStatus int
		checkContext   bool
		expectedID     string
	}{
		{
			name:           "Missing Authorization Header",
			authHeader:     "",
			setupMock:      func() {},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Invalid Authorization Format",
			authHeader:     "Basic 12345",
			setupMock:      func() {},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Valid JWT",
			authHeader:     "Bearer " + createToken(secret, "mfg-jwt", time.Hour),
			setupMock:      func() {},
			expectedStatus: http.StatusOK,
			checkContext:   true,
			expectedID:     "mfg-jwt",
		},
		{
			name:       "Valid API Key",
			authHeader: "Bearer my-api-key",
			setupMock: func() {
				mockRepo.On("ValidateKey", mock.Anything, createKeyHash("my-api-key")).Return("mfg-api", true, nil)
			},
			expectedStatus: http.StatusOK,
			checkContext:   true,
			expectedID:     "mfg-api",
		},
		{
			name:       "Invalid API Key",
			authHeader: "Bearer wrong-key",
			setupMock: func() {
				mockRepo.On("ValidateKey", mock.Anything, createKeyHash("wrong-key")).Return("", false, nil)
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:       "Redis Error",
			authHeader: "Bearer error-key",
			setupMock: func() {
				mockRepo.On("ValidateKey", mock.Anything, createKeyHash("error-key")).Return("", false, errors.New("redis down"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset mock
			mockRepo.ExpectedCalls = nil
			mockRepo.Calls = nil
			tt.setupMock()

			// Create a dummy handler that checks the context
			nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.checkContext {
					id, ok := r.Context().Value(ManufacturerIDKey).(string)
					assert.True(t, ok)
					assert.Equal(t, tt.expectedID, id)
				}
				w.WriteHeader(http.StatusOK)
			})

			req := httptest.NewRequest("GET", "/", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			rec := httptest.NewRecorder()

			middleware(nextHandler).ServeHTTP(rec, req)

			assert.Equal(t, tt.expectedStatus, rec.Code)
			mockRepo.AssertExpectations(t)
		})
	}
}
