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
	"github.com/redis/go-redis/v9"
)

type RedisAuthRepository struct {
	client *redis.Client
}

// Ensure interface compliance
var _ ports.AuthRepository = (*RedisAuthRepository)(nil)

func NewRedisAuthRepository(client *redis.Client) *RedisAuthRepository {
	return &RedisAuthRepository{client: client}
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

	// Key exists, return the TenantID (which is stored as the value)
	return val, true, nil
}
