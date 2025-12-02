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
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/httprate"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	// 1. Config
	cfg := config.Load()
	log := logger.New(cfg.LogLevel, cfg.IsProduction())

	// 2. Infrastructure
	ctx := context.Background()
	dbPool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Error("DB Connection failed", "error", err)
		return
	}
	defer dbPool.Close()

	// 2a. Initialize Cache
	redisClient := cache.NewRedisClient(cfg.RedisAddr)
	redisStore := cache.NewRedisStore(redisClient)
	authRepo := cache.NewRedisAuthRepository(redisClient, dbPool)

	// 2b. Warmup Cache (Load API Keys)
	log.Info("Warming up auth cache...")
	if err := authRepo.Warmup(ctx); err != nil {
		log.Warn("Failed to warmup auth cache", "error", err)
		// We don't exit fatal here, because maybe DB is empty or partial failure.
		// But in a strict environment, maybe we should.
	}

	// Initialize Blob Storage
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

	// Initialize Event Bus (Resolver doesn't publish, but service requires it)
	eventBus := bus.NewRedisEventBus(cfg.RedisAddr)

	// 3. Wiring (Identical to Ingest, but we use different handlers)
	repo := postgres.NewPassportRepository(dbPool)
	svc, err := service.NewPassportService(repo, redisStore, blobStore, eventBus, log)
	if err != nil {
		log.Error("Failed to initialize service", "error", err)
		return
	}

	handler := rest.NewResolverHandler(svc, authRepo, log, cfg)

	// 4. Router
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	// Rate Limiting: 100 requests per minute per IP
	// This protects the application layer from simple flooding.
	// For massive DDoS, rely on Cloudflare/WAF.
	r.Use(httprate.LimitByIP(100, 1*time.Minute))

	// CORS (Dev Only)
	if !cfg.IsProduction() {
		log.Info("Enabling CORS for Development")
		r.Use(cors.Handler(cors.Options{
			AllowedOrigins:   []string{"http://localhost:3000"},
			AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
			ExposedHeaders:   []string{"Link"},
			AllowCredentials: true,
			MaxAge:           300, // Maximum value not ignored by any of major browsers
		}))
	}

	// Public Routes
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	handler.RegisterResolverRoutes(r)

	// 5. Start
	port := ":8081" // Note: Different port than Ingest (8080)
	log.Info("TraceApi Resolver Server starting", "port", port)
	if err := http.ListenAndServe(port, r); err != nil {
		log.Error("Server failed", "error", err)
	}
}
