/*
 * Copyright (c) 2025 Alessandro Faranda Gancio (dba TraceApi)
 *
 * This source code is licensed under the Business Source License 1.1.
 *
 * Change Date: 2027-11-21
 * Change License: AGPL-3.0
 */

package main

import (
	"context"
	"net/http"
	"time"

	"github.com/TraceApi/api-core/internal/config"
	"github.com/TraceApi/api-core/internal/core/service"
	"github.com/TraceApi/api-core/internal/platform/cache"
	"github.com/TraceApi/api-core/internal/platform/logger"
	"github.com/TraceApi/api-core/internal/platform/storage/postgres"
	"github.com/TraceApi/api-core/internal/transport/rest"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
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

	redisStore := cache.NewRedisStore(cfg.RedisAddr)

	// 3. Wiring (Identical to Ingest, but we use different handlers)
	repo := postgres.NewPassportRepository(dbPool)
	svc, err := service.NewPassportService(repo, redisStore, log)
	if err != nil {
		log.Error("Failed to initialize service", "error", err)
		return
	}

	handler := rest.NewResolverHandler(svc, log)

	// 4. Router
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	// Mount Public Routes
	handler.RegisterResolverRoutes(r)

	// 5. Start
	port := ":8081" // Note: Different port than Ingest (8080)
	log.Info("TraceApi Resolver Server starting", "port", port)
	if err := http.ListenAndServe(port, r); err != nil {
		log.Error("Server failed", "error", err)
	}
}
