/*
 * Copyright (c) 2025 Alessandro Faranda Gancio (dba TraceApi)
 *
 * This source code is licensed under the Business Source License 1.1.
 *
 * Change Date: 2027-11-28
 * Change License: AGPL-3.0
 */

package ports

import (
	"context"

	"github.com/TraceApi/api-core/internal/core/domain" // Adjust module path if needed
	"github.com/google/uuid"
)

type PassportRepository interface {
	// Save creates or updates a passport
	Save(ctx context.Context, passport *domain.Passport) error

	// Update updates an existing passport (status, hash, published_at, storage_location)
	Update(ctx context.Context, passport *domain.Passport) error

	// GetByID retrieves a single passport
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Passport, error)

	// FindByCategory retrieves a page of passports (Basic pagination)
	FindByCategory(ctx context.Context, category domain.ProductCategory, limit, offset int) ([]*domain.Passport, error)
}
