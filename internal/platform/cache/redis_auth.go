/*
 * Copyright (c) 2025 Alessandro Faranda Gancio (dba TraceApi)
 *
 * This source code is licensed under the Business Source License 1.1.
 *
 * Change Date: 2027-11-28
 * Change License: AGPL-3.0
 */

package cache

import (
	"context"
	"fmt"

	"github.com/TraceApi/api-core/internal/core/ports"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type RedisAuthRepository struct {
	client *redis.Client
	db     *pgxpool.Pool
}

// Ensure interface compliance
var _ ports.AuthRepository = (*RedisAuthRepository)(nil)

func NewRedisAuthRepository(client *redis.Client, db *pgxpool.Pool) *RedisAuthRepository {
	return &RedisAuthRepository{client: client, db: db}
}

func (r *RedisAuthRepository) ValidateKey(ctx context.Context, apiKeyHash string) (string, bool, error) {
	// Key format: "auth:apikey:{hash}" -> value: "{tenant_id}"
	redisKey := fmt.Sprintf("auth:apikey:%s", apiKeyHash)

	val, err := r.client.Get(ctx, redisKey).Result()
	if err == redis.Nil {
		// Key does not exist = Invalid
		return "", false, nil
	}
	if err != nil {
		// System error (Redis down)
		return "", false, err
	}

	// Key exists, return the TenantID
	return val, true, nil
}

// Warmup loads all active API keys from Postgres into Redis.
// This should be called on service startup.
func (r *RedisAuthRepository) Warmup(ctx context.Context) error {
	query := `SELECT key_hash, tenant_id FROM api_keys WHERE expires_at > NOW()`
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to query api_keys: %w", err)
	}
	defer rows.Close()

	pipeline := r.client.Pipeline()
	count := 0

	for rows.Next() {
		var hash, tenantID string
		if err := rows.Scan(&hash, &tenantID); err != nil {
			return fmt.Errorf("failed to scan api_key: %w", err)
		}

		redisKey := fmt.Sprintf("auth:apikey:%s", hash)
		// We don't set an expiry (0) because these are long-lived keys.
		// Or we could set it to the actual expiry time from DB, but 0 is simpler for now.
		pipeline.Set(ctx, redisKey, tenantID, 0)
		count++
	}

	if count > 0 {
		if _, err := pipeline.Exec(ctx); err != nil {
			return fmt.Errorf("failed to execute redis pipeline: %w", err)
		}
	}

	return nil
}

func (r *RedisAuthRepository) GetTenantState(ctx context.Context, tenantID string) (string, error) {
	key := fmt.Sprintf("tenant:state:%s", tenantID)
	val, err := r.client.Get(ctx, key).Result()

	if err == redis.Nil {
		// If no state key exists, assume ACTIVE
		return "ACTIVE", nil
	}
	return val, err
}
