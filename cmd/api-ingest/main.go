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
	"log"
	"net/http"
	"os"
	"time"

	"github.com/TraceApi/api-core/internal/core/service"
	"github.com/TraceApi/api-core/internal/platform/cache"
	"github.com/TraceApi/api-core/internal/platform/storage/postgres"
	"github.com/TraceApi/api-core/internal/transport/rest"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	// 1. Configuration (Env Vars)
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://trace_user:trace_password@localhost:5432/trace_core?sslmode=disable"
	}

	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "trace_cache:6379"
	}

	// 2. Database Connection
	ctx := context.Background()
	dbPool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}
	defer dbPool.Close()

	// 2a. Initialize Cache
	redisStore := cache.NewRedisStore(redisAddr)

	// 3. Dependency Injection (Wiring)
	// Repo -> Service -> Handler
	passportRepo := postgres.NewPassportRepository(dbPool)

	// Inject Cache into Service
	passportSvc, err := service.NewPassportService(passportRepo, redisStore)
	if err != nil {
		log.Fatalf("Failed to initialize service: %v", err)
	}

	passportHandler := rest.NewPassportHandler(passportSvc)

	// 4. Router Setup
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	// Mount API routes
	r.Route("/v1", func(r chi.Router) {
		passportHandler.RegisterRoutes(r)
	})

	// 5. Start Server
	port := ":8080"
	log.Printf("TraceApi Ingest Server starting on %s", port)
	if err := http.ListenAndServe(port, r); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
