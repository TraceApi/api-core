/*
 * Copyright (c) 2025 Alessandro Faranda Gancio (dba TraceApi)
 *
 * This source code is licensed under the Business Source License 1.1.
 *
 * Change Date: 2027-11-28
 * Change License: AGPL-3.0
 */

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
			attributes, created_at, updated_at, published_at, immutability_hash, storage_location
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
		)
		ON CONFLICT (id) DO UPDATE SET
			status = EXCLUDED.status,
			attributes = EXCLUDED.attributes,
			updated_at = EXCLUDED.updated_at,
			published_at = EXCLUDED.published_at,
			immutability_hash = EXCLUDED.immutability_hash,
			storage_location = EXCLUDED.storage_location;
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
		p.StorageLocation,
	)
	return err
}

func (r *PostgresRepository) Update(ctx context.Context, p *domain.Passport) error {
	query := `
		UPDATE passports SET
			status = $2,
			immutability_hash = $3,
			published_at = $4,
			storage_location = $5,
			updated_at = $6,
			attributes = $7
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query,
		p.ID,
		p.Status,
		p.ImmutabilityHash,
		p.PublishedAt,
		p.StorageLocation,
		time.Now(),
		p.Attributes,
	)
	return err
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

func (r *PostgresRepository) FindByManufacturer(ctx context.Context, manufacturerID string) ([]*domain.Passport, error) {
	query := `
		SELECT id, product_category, status, manufacturer_id, manufacturer_name, attributes, created_at, updated_at, published_at
		FROM passports
		WHERE manufacturer_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(ctx, query, manufacturerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var passports []*domain.Passport
	for rows.Next() {
		var p domain.Passport
		var publishedAt *time.Time
		if err := rows.Scan(&p.ID, &p.ProductCategory, &p.Status, &p.ManufacturerID, &p.ManufacturerName, &p.Attributes, &p.CreatedAt, &p.UpdatedAt, &publishedAt); err != nil {
			return nil, err
		}
		p.PublishedAt = publishedAt
		passports = append(passports, &p)
	}

	return passports, nil
}
