/*
 * Copyright (c) 2025 TraceApi
 *
 * This source code is licensed under the Business Source License 1.1.
 *
 * Change Date: 2029-11-20
 * Change License: AGPL-3.0
 */

package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	_ "embed"

	"github.com/TraceApi/api-core/internal/core/domain"
	"github.com/TraceApi/api-core/internal/core/ports"
	"github.com/TraceApi/api-core/internal/platform/cache"
	"github.com/google/uuid"
	"github.com/santhosh-tekuri/jsonschema/v5"
)

// Embed the schemas directly into the Go binary
//
//go:embed schemas/payloads/battery.json
var batterySchemaRaw string

//go:embed schemas/payloads/textile.json
var textileSchemaRaw string

type passportService struct {
	repo     ports.PassportRepository
	cache    *cache.RedisStore
	compiler *jsonschema.Compiler
	schemas  map[domain.ProductCategory]*jsonschema.Schema
}

// Ensure interface implementation
var _ ports.PassportService = (*passportService)(nil)

func NewPassportService(repo ports.PassportRepository, cache *cache.RedisStore) (ports.PassportService, error) {
	compiler := jsonschema.NewCompiler()
	compiler.Draft = jsonschema.Draft2020

	// Register and Compile Schemas
	if err := compiler.AddResource("battery.json", strings.NewReader(batterySchemaRaw)); err != nil {
		return nil, fmt.Errorf("failed to add battery schema: %w", err)
	}
	if err := compiler.AddResource("textile.json", strings.NewReader(textileSchemaRaw)); err != nil {
		return nil, fmt.Errorf("failed to add textile schema: %w", err)
	}

	batterySchema, err := compiler.Compile("battery.json")
	if err != nil {
		return nil, fmt.Errorf("failed to compile battery schema: %w", err)
	}

	textileSchema, err := compiler.Compile("textile.json")
	if err != nil {
		return nil, fmt.Errorf("failed to compile textile schema: %w", err)
	}

	return &passportService{
		repo:     repo,
		cache:    cache,
		compiler: compiler,
		schemas: map[domain.ProductCategory]*jsonschema.Schema{
			domain.CategoryBattery: batterySchema,
			domain.CategoryTextile: textileSchema,
		},
	}, nil
}

func (s *passportService) CreatePassport(ctx context.Context, manufacturerID string, category domain.ProductCategory, payload []byte) (*domain.Passport, error) {
	// 1. Idempotency Check
	// Generate a hash of the raw payload + category + manufacturer
	hasher := sha256.New()
	hasher.Write([]byte(manufacturerID))
	hasher.Write([]byte(category))
	hasher.Write(payload)
	payloadHash := hex.EncodeToString(hasher.Sum(nil))

	// Check Redis for existing hash
	if existingIDStr, err := s.cache.GetIdempotency(ctx, payloadHash); err == nil {
		// Cache HIT! The passport exists.
		// Parse the UUID string back to UUID
		uid, parseErr := uuid.Parse(existingIDStr)
		if parseErr == nil {
			// Fetch the full object from DB so the API response is identical
			existingPassport, dbErr := s.repo.GetByID(ctx, uid)
			if dbErr == nil {
				return existingPassport, nil
			}
		}
		// If parsing failed or DB lookup failed, we fall through and recreate (safe fallback)
	}

	// 2. Schema Validation
	schema, exists := s.schemas[category]
	if !exists {
		return nil, fmt.Errorf("unsupported product category: %s", category)
	}

	var jsonInterface interface{}
	if err := json.Unmarshal(payload, &jsonInterface); err != nil {
		return nil, fmt.Errorf("invalid JSON format: %w", err)
	}

	if err := schema.Validate(jsonInterface); err != nil {
		return nil, fmt.Errorf("schema validation failed: %#v", err)
	}

	// 3. Construct Domain Entity
	now := time.Now().UTC()
	passport := &domain.Passport{
		ID:               uuid.New(),
		ProductCategory:  category,
		Status:           domain.StatusDraft,
		ManufacturerID:   manufacturerID,
		ManufacturerName: "Unknown Manufacturer", // Placeholder until Auth Service
		Attributes:       json.RawMessage(payload),
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	// 4. Save to Repository
	if err := s.repo.Save(ctx, passport); err != nil {
		return nil, fmt.Errorf("failed to persist passport: %w", err)
	}

	// 5. Save to Idempotency Cache
	// We do this LAST. If it fails, we log it but don't fail the request.
	// Ideally, use a logger here. For now, we suppress the error.
	_ = s.cache.SetIdempotency(ctx, payloadHash, passport.ID.String())

	return passport, nil
}

func (s *passportService) GetPassport(ctx context.Context, id uuid.UUID) (*domain.Passport, error) {
	cacheKey := fmt.Sprintf("passport:%s", id.String())

	// 1. FAST PATH: Check Redis
	cachedJSON, err := s.cache.Get(ctx, cacheKey)
	if err == nil {
		// Cache Hit! Unmarshal and return.
		var p domain.Passport
		if jsonErr := json.Unmarshal([]byte(cachedJSON), &p); jsonErr == nil {
			return &p, nil
		}
		// If unmarshal fails, we ignore the cache and hit the DB (Auto-repair)
	}

	// 2. SLOW PATH: Hit Postgres
	passport, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// 3. FILL CACHE: Save for next time
	// We cache for 1 hour (or longer, since passports are immutable-ish)
	if jsonBytes, jsonErr := json.Marshal(passport); jsonErr == nil {
		// Run in goroutine so we don't block the response
		go func() {
			// Create a detached context so the cache set doesn't fail if the HTTP request cancels
			_ = s.cache.Set(context.Background(), cacheKey, string(jsonBytes), 1*time.Hour)
		}()
	}

	return passport, nil
}
