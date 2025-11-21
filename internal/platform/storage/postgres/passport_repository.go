package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/TraceApi/api-core/internal/core/domain"
	"github.com/TraceApi/api-core/internal/core/ports"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct {
	db *pgxpool.Pool
}

// Ensure we implement the interface
var _ ports.PassportRepository = (*PostgresRepository)(nil)

func NewPassportRepository(db *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) Save(ctx context.Context, p *domain.Passport) error {
	query := `
		INSERT INTO passports (
			id, product_category, status, manufacturer_id, manufacturer_name, 
			attributes, created_at, updated_at, published_at, immutability_hash
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10
		)
		ON CONFLICT (id) DO UPDATE SET
			status = EXCLUDED.status,
			attributes = EXCLUDED.attributes,
			updated_at = EXCLUDED.updated_at,
			published_at = EXCLUDED.published_at,
			immutability_hash = EXCLUDED.immutability_hash;
	`

	// Handle nullable PublishedAt
	var publishedAt *time.Time
	if p.PublishedAt != nil {
		publishedAt = p.PublishedAt
	}

	_, err := r.db.Exec(ctx, query,
		p.ID,
		p.ProductCategory,
		p.Status,
		p.ManufacturerID,
		p.ManufacturerName,
		p.Attributes,
		p.CreatedAt,
		p.UpdatedAt,
		publishedAt,
		p.ImmutabilityHash,
	)

	if err != nil {
		return fmt.Errorf("failed to save passport: %w", err)
	}
	return nil
}

func (r *PostgresRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Passport, error) {
	query := `
		SELECT id, product_category, status, manufacturer_id, manufacturer_name, 
		       attributes, created_at, updated_at, published_at, immutability_hash
		FROM passports
		WHERE id = $1
	`

	var p domain.Passport
	var publishedAt *time.Time

	err := r.db.QueryRow(ctx, query, id).Scan(
		&p.ID,
		&p.ProductCategory,
		&p.Status,
		&p.ManufacturerID,
		&p.ManufacturerName,
		&p.Attributes,
		&p.CreatedAt,
		&p.UpdatedAt,
		&publishedAt,
		&p.ImmutabilityHash,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("passport not found: %w", err)
		}
		return nil, fmt.Errorf("database error: %w", err)
	}

	p.PublishedAt = publishedAt
	return &p, nil
}

func (r *PostgresRepository) FindByCategory(ctx context.Context, category domain.ProductCategory, limit, offset int) ([]*domain.Passport, error) {
	query := `
		SELECT id, product_category, status, manufacturer_id, manufacturer_name, attributes
		FROM passports
		WHERE product_category = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.Query(ctx, query, category, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var passports []*domain.Passport
	for rows.Next() {
		var p domain.Passport
		if err := rows.Scan(&p.ID, &p.ProductCategory, &p.Status, &p.ManufacturerID, &p.ManufacturerName, &p.Attributes); err != nil {
			return nil, err
		}
		passports = append(passports, &p)
	}

	return passports, nil
}
