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

	"github.com/TraceApi/api-core/internal/core/domain"
	"github.com/google/uuid"
)

type PassportService interface {
	// CreatePassport takes raw JSON input and the intended category.
	// It returns the created Passport (with ID) or a validation error.
	CreatePassport(ctx context.Context, manufacturerID string, manufacturerName string, category domain.ProductCategory, payload []byte) (*domain.Passport, error)

	GetPassport(ctx context.Context, id uuid.UUID) (*domain.Passport, error)

	PublishPassport(ctx context.Context, id uuid.UUID) (*domain.Passport, error)

	ListPassports(ctx context.Context, manufacturerID string) ([]*domain.Passport, error)

	UpdatePassport(ctx context.Context, id uuid.UUID, manufacturerID string, payload []byte) (*domain.Passport, error)
}
