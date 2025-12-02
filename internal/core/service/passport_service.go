/*
 * Copyright (c) 2025 Alessandro Faranda Gancio (dba TraceApi)
 *
 * This source code is licensed under the Business Source License 1.1.
 *
 * Change Date: 2027-11-28
 * Change License: AGPL-3.0
 */

package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	_ "embed"

	"github.com/TraceApi/api-core/internal/core/domain"
	"github.com/TraceApi/api-core/internal/core/ports"
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
	repo             ports.PassportRepository
	cache            ports.CacheRepository
	blobStore        ports.BlobStorage
	eventBus         ports.EventBus
	compiler         *jsonschema.Compiler
	schemas          map[domain.ProductCategory]*jsonschema.Schema
	restrictedFields map[domain.ProductCategory][]string
	log              *slog.Logger
}

// Ensure interface implementation
var _ ports.PassportService = (*passportService)(nil)

func NewPassportService(repo ports.PassportRepository, cache ports.CacheRepository, blobStore ports.BlobStorage, eventBus ports.EventBus, log *slog.Logger) (ports.PassportService, error) {
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

	// Parse Restricted Fields
	restrictedFields := make(map[domain.ProductCategory][]string)

	batRestricted, err := parseRestrictedFields(batterySchemaRaw)
	if err != nil {
		return nil, fmt.Errorf("failed to parse restricted fields for battery: %w", err)
	}
	restrictedFields[domain.CategoryBattery] = batRestricted

	texRestricted, err := parseRestrictedFields(textileSchemaRaw)
	if err != nil {
		return nil, fmt.Errorf("failed to parse restricted fields for textile: %w", err)
	}
	restrictedFields[domain.CategoryTextile] = texRestricted

	return &passportService{
		repo:      repo,
		cache:     cache,
		blobStore: blobStore,
		eventBus:  eventBus,
		compiler:  compiler,
		schemas: map[domain.ProductCategory]*jsonschema.Schema{
			domain.CategoryBattery: batterySchema,
			domain.CategoryTextile: textileSchema,
		},
		restrictedFields: restrictedFields,
		log:              log,
	}, nil
}

func parseRestrictedFields(rawSchema string) ([]string, error) {
	var schema struct {
		Properties map[string]struct {
			Access string `json:"access"`
		} `json:"properties"`
	}
	if err := json.Unmarshal([]byte(rawSchema), &schema); err != nil {
		return nil, err
	}
	var restricted []string
	for key, prop := range schema.Properties {
		if prop.Access == "restricted" {
			restricted = append(restricted, key)
		}
	}
	return restricted, nil
}

func (s *passportService) CreatePassport(ctx context.Context, manufacturerID string, manufacturerName string, category domain.ProductCategory, payload []byte) (*domain.Passport, error) {
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
		s.log.Warn("unsupported product category", "category", category)
		return nil, fmt.Errorf("%w: %s", domain.ErrInvalidInput, category)
	}

	var jsonInterface interface{}
	if err := json.Unmarshal(payload, &jsonInterface); err != nil {
		s.log.Warn("invalid json format", "error", err)
		return nil, fmt.Errorf("%w: invalid JSON", domain.ErrInvalidInput)
	}

	if err := schema.Validate(jsonInterface); err != nil {
		s.log.Warn("schema validation failed", "error", err)
		return nil, fmt.Errorf("%w: schema validation failed", domain.ErrInvalidInput)
	}

	// 3. Construct Domain Entity
	now := time.Now().UTC()
	passport := &domain.Passport{
		ID:               uuid.New(),
		ProductCategory:  category,
		Status:           domain.StatusDraft,
		ManufacturerID:   manufacturerID,
		ManufacturerName: manufacturerName,
		Attributes:       json.RawMessage(payload),
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	// 4. Save to Repository
	if err := s.repo.Save(ctx, passport); err != nil {
		s.log.Error("failed to persist passport", "error", err)
		return nil, fmt.Errorf("%w: failed to save", domain.ErrInternal)
	}

	// 5. Save to Idempotency Cache
	// We do this LAST. If it fails, we log it but don't fail the request.
	if err := s.cache.SetIdempotency(ctx, payloadHash, passport.ID.String()); err != nil {
		s.log.Warn("failed to set idempotency key", "error", err)
	}

	// 6. Publish Event
	event := struct {
		TenantID   string    `json:"tenant_id"`
		PassportID string    `json:"passport_id"`
		Timestamp  time.Time `json:"timestamp"`
	}{
		TenantID:   manufacturerID,
		PassportID: passport.ID.String(),
		Timestamp:  time.Now().UTC(),
	}

	if err := s.eventBus.Publish(ctx, "events:passport_created", event); err != nil {
		s.log.Error("failed to publish passport_created event", "error", err)
	}

	return passport, nil
}

func (s *passportService) GetPassport(ctx context.Context, id uuid.UUID) (*domain.Passport, error) {
	cacheKey := fmt.Sprintf("passport:%s", id.String())
	var passport *domain.Passport

	// 1. FAST PATH: Check Redis
	cachedJSON, err := s.cache.Get(ctx, cacheKey)
	if err == nil {
		var p domain.Passport
		if jsonErr := json.Unmarshal([]byte(cachedJSON), &p); jsonErr == nil {
			passport = &p
			s.log.Debug("Cache Hit", "id", id)
		}
	}

	// 2. SLOW PATH: Hit Postgres (if not in cache)
	if passport == nil {
		p, err := s.repo.GetByID(ctx, id)
		if err != nil {
			return nil, err
		}
		passport = p

		// 3. FILL CACHE: Save for next time (Full Data)
		// We cache for 1 hour (or longer, since passports are immutable-ish)
		if jsonBytes, jsonErr := json.Marshal(passport); jsonErr == nil {
			// Run in goroutine so we don't block the response
			go func() {
				// Create a detached context so the cache set doesn't fail if the HTTP request cancels
				_ = s.cache.Set(context.Background(), cacheKey, string(jsonBytes), 1*time.Hour)
			}()
		}
	}

	// 4. FILTERING (Public vs Restricted)
	// This MUST run after retrieval (Cache OR DB) to ensure we don't leak secrets
	viewContext, _ := ctx.Value(domain.ViewContextKey).(string)
	viewerTenantID, _ := ctx.Value(domain.ViewerTenantIDKey).(string)

	// Strict Ownership Check:
	// Even if authenticated (Restricted Context), you can only see restricted data
	// if you are the Manufacturer of this passport.
	isOwner := (viewerTenantID == passport.ManufacturerID)

	if viewContext != domain.ViewContextRestricted || !isOwner {
		s.filterAttributes(passport)
	}

	return passport, nil
}

func (s *passportService) filterAttributes(passport *domain.Passport) {
	restricted, ok := s.restrictedFields[passport.ProductCategory]
	if !ok || len(restricted) == 0 {
		return
	}

	var attrs map[string]interface{}
	if err := json.Unmarshal(passport.Attributes, &attrs); err != nil {
		s.log.Warn("failed to unmarshal attributes for filtering", "error", err)
		return
	}

	for _, field := range restricted {
		delete(attrs, field)
	}

	if filtered, err := json.Marshal(attrs); err == nil {
		passport.Attributes = json.RawMessage(filtered)
	}
}

func (s *passportService) PublishPassport(ctx context.Context, id uuid.UUID) (*domain.Passport, error) {
	// 1. Fetch Passport
	passport, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch passport: %w", err)
	}

	// 2. Check if already published
	if passport.Status == domain.StatusPublished {
		return nil, domain.ErrPassportAlreadyPublished
	}

	// 3. Marshal Attributes
	payloadBytes, err := json.Marshal(passport.Attributes)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal attributes: %w", err)
	}

	// 4. Calculate SHA-256 Hash
	hash := sha256.Sum256(payloadBytes)
	hashString := hex.EncodeToString(hash[:])

	// 5. Upload to BlobStorage
	key := fmt.Sprintf("passports/%s.json", passport.ID.String())
	s3URL, err := s.blobStore.UploadJSON(ctx, "passports", key, payloadBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to upload to blob storage: %w", err)
	}

	// 6. Update Passport Struct
	passport.Status = domain.StatusPublished
	passport.ImmutabilityHash = hashString
	passport.StorageLocation = s3URL
	now := time.Now()
	passport.PublishedAt = &now

	// 7. Save to Repo
	if err := s.repo.Update(ctx, passport); err != nil {
		return nil, fmt.Errorf("failed to save published passport: %w", err)
	}

	// 8. Invalidate Cache (Force next read to hit DB)
	cacheKey := fmt.Sprintf("passport:%s", id.String())
	go func() {
		_ = s.cache.Delete(context.Background(), cacheKey)
	}()

	return passport, nil
}
