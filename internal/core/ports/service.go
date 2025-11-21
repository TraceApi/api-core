package ports

import (
	"context"

	"github.com/TraceApi/api-core/internal/core/domain"
)

type PassportService interface {
	// CreatePassport takes raw JSON input and the intended category.
	// It returns the created Passport (with ID) or a validation error.
	CreatePassport(ctx context.Context, manufacturerID string, category domain.ProductCategory, payload []byte) (*domain.Passport, error)
}
