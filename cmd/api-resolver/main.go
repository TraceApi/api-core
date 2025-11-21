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

	"github.com/TraceApi/api-core/internal/core/service"
	"github.com/TraceApi/api-core/internal/platform/cache"
	"github.com/TraceApi/api-core/internal/platform/storage/postgres"
	"github.com/TraceApi/api-core/internal/transport/rest"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	// 1. Config
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://trace_user:trace_password@localhost:5432/trace_core?sslmode=disable"
	}
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	// 2. Infrastructure
	ctx := context.Background()
	dbPool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("DB Connection failed: %v", err)
	}
	defer dbPool.Close()

	redisStore := cache.NewRedisStore(redisAddr)

	// 3. Wiring (Identical to Ingest, but we use different handlers)
	repo := postgres.NewPassportRepository(dbPool)
	svc, _ := service.NewPassportService(repo, redisStore) // Error ignored for brevity in snippet

	handler := rest.NewResolverHandler(svc)

	// 4. Router
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Mount Public Routes
	handler.RegisterResolverRoutes(r)

	// 5. Start
	port := ":8081" // Note: Different port than Ingest (8080)
	log.Printf("TraceApi Resolver Server starting on %s", port)
	if err := http.ListenAndServe(port, r); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
