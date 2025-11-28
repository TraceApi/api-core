//go:build integration
// +build integration

/*
 * Copyright (c) 2025 Alessandro Faranda Gancio (dba TraceApi)
 *
 * This source code is licensed under the Business Source License 1.1.
 *
 * Change Date: 2027-11-28
 * Change License: AGPL-3.0
 */

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/TraceApi/api-core/internal/config"
	"github.com/TraceApi/api-core/internal/core/domain"
	"github.com/TraceApi/api-core/internal/core/service"
	"github.com/TraceApi/api-core/internal/platform/bus"
	"github.com/TraceApi/api-core/internal/platform/cache"
	"github.com/TraceApi/api-core/internal/platform/logger"
	"github.com/TraceApi/api-core/internal/platform/storage/postgres"
	"github.com/TraceApi/api-core/internal/platform/storage/s3"
	"github.com/TraceApi/api-core/internal/transport/rest"
	authMiddleware "github.com/TraceApi/api-core/internal/transport/rest/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupIntegrationServer spins up a real HTTP server connected to the local Docker infrastructure.
// It returns the test server and a cleanup function.
func setupIntegrationServer(t *testing.T) (*httptest.Server, func()) {
	cfg := config.Load()
	log := logger.New(cfg.LogLevel, false)

	// 1. Database Connection
	ctx := context.Background()
	dbPool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	require.NoError(t, err, "Failed to connect to database")

	// 2. Cache
	redisStore := cache.NewRedisStore(cfg.RedisAddr)

	// 2b. Event Bus
	eventBus := bus.NewRedisEventBus(cfg.RedisAddr)

	// 3. Blob Storage
	blobStore, err := s3.NewBlobStore(ctx, s3.Config{
		Endpoint:  cfg.S3Endpoint,
		Region:    cfg.S3Region,
		AccessKey: cfg.S3AccessKey,
		SecretKey: cfg.S3SecretKey,
	})
	require.NoError(t, err, "Failed to initialize blob store")

	// 4. Wiring
	passportRepo := postgres.NewPassportRepository(dbPool)
	passportSvc, err := service.NewPassportService(passportRepo, redisStore, blobStore, eventBus, log)
	require.NoError(t, err, "Failed to initialize service")

	passportHandler := rest.NewPassportHandler(passportSvc, log)

	// 5. Router
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	r.Group(func(r chi.Router) {
		r.Use(authMiddleware.HybridAuthMiddleware(cfg.JWTSecret, redisStore, log))
		passportHandler.RegisterRoutes(r)
	})

	ts := httptest.NewServer(r)

	cleanup := func() {
		ts.Close()
		dbPool.Close()
	}

	return ts, cleanup
}

// TestFullPassportLifecycle verifies the entire flow:
// 1. Generate Token
// 2. Create Passport (Ingest API)
// 3. Resolve Passport (Resolver API - simulated via direct DB check for now or separate client)
func TestFullPassportLifecycle(t *testing.T) {
	// 1. Setup Server (Grey Box)
	ts, cleanup := setupIntegrationServer(t)
	defer cleanup()

	cfg := config.Load()

	// 2. Generate a valid JWT
	token := generateTestToken(cfg.JWTSecret)

	// 3. Define the Payload
	payload := map[string]interface{}{
		"batteryModel":  "Integration Test Pack",
		"chemistry":     "LITHIUM_IRON_PHOSPHATE",
		"ratedCapacity": 100,
		"carbonFootprint": map[string]interface{}{
			"totalCarbonFootprint": 50.5,
			"shareOfRenewables":    90,
		},
		"materialComposition": []map[string]interface{}{
			{"material": "Lithium", "massPercentage": 5},
		},
	}
	payloadBytes, _ := json.Marshal(payload)

	// 4. Execute Request against the TEST server
	client := ts.Client()
	req, _ := http.NewRequest("POST", fmt.Sprintf("%s/passports?category=BATTERY_INDUSTRIAL", ts.URL), bytes.NewBuffer(payloadBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// 5. Assert Creation
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var passport domain.Passport
	err = json.NewDecoder(resp.Body).Decode(&passport)
	require.NoError(t, err)
	assert.NotEmpty(t, passport.ID)
	assert.Equal(t, domain.StatusDraft, passport.Status)

	t.Logf("Successfully created passport: %s", passport.ID)
}

func generateTestToken(secret string) string {
	claims := jwt.MapClaims{
		"sub":             "integration-test-mfg",
		"manufacturer_id": "integration-test-mfg",
		"exp":             time.Now().Add(1 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	s, _ := token.SignedString([]byte(secret))
	return s
}
