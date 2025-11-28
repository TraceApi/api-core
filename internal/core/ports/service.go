package ports

import (
	"context"

	"github.com/TraceApi/api-core/internal/core/domain"
	"github.com/google/uuid"
)

type PassportService interface {
	// CreatePassport takes raw JSON input and the intended category.
	// It returns the created Passport (with ID) or a validation error.
	CreatePassport(ctx context.Context, manufacturerID string, category domain.ProductCategory, payload []byte) (*domain.Passport, error)

	GetPassport(ctx context.Context, id uuid.UUID) (*domain.Passport, error)

	PublishPassport(ctx context.Context, id uuid.UUID) (*domain.Passport, error)
}
