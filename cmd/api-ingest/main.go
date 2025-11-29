/*
 * Copyright (c) 2025 Alessandro Faranda Gancio (dba TraceApi)
 *
 * This source code is licensed under the Business Source License 1.1.
 *
 * Change Date: 2027-11-28
 * Change License: AGPL-3.0
 */

package main

import (
	"context"
	"net/http"
	"time"

	"github.com/TraceApi/api-core/internal/config"
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
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	// 1. Configuration
	cfg := config.Load()
	log := logger.New(cfg.LogLevel, cfg.IsProduction())

	// 2. Database Connection
	ctx := context.Background()
	dbPool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Error("Unable to connect to database", "error", err)
		return
	}
	defer dbPool.Close()

	// 2a. Initialize Cache
	redisClient := cache.NewRedisClient(cfg.RedisAddr)
	redisStore := cache.NewRedisStore(redisClient)
	authRepo := cache.NewRedisAuthRepository(redisClient)

	// 2b. Initialize Blob Storage
	blobStore, err := s3.NewBlobStore(ctx, s3.Config{
		Endpoint:  cfg.S3Endpoint,
		Region:    cfg.S3Region,
		AccessKey: cfg.S3AccessKey,
		SecretKey: cfg.S3SecretKey,
	})
	if err != nil {
		log.Error("Failed to initialize blob store", "error", err)
		return
	}

	// 2c. Initialize Event Bus
	eventBus := bus.NewRedisEventBus(cfg.RedisAddr)

	// 3. Dependency Injection (Wiring)
	// Repo -> Service -> Handler
	passportRepo := postgres.NewPassportRepository(dbPool)

	// Inject Cache into Service
	passportSvc, err := service.NewPassportService(passportRepo, redisStore, blobStore, eventBus, log)
	if err != nil {
		log.Error("Failed to initialize service", "error", err)
		return
	}

	passportHandler := rest.NewPassportHandler(passportSvc, log)

	// 4. Router Setup
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	// Public Routes
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Protected Routes
	r.Group(func(r chi.Router) {
		r.Use(authMiddleware.HybridAuthMiddleware(cfg.JWTSecret, authRepo, log))
		passportHandler.RegisterRoutes(r)
	})

	log.Info("Starting server", "port", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, r); err != nil {
		log.Error("Server failed", "error", err)
	}
}
