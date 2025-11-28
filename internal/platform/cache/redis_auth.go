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

	"github.com/redis/go-redis/v9"
)

// ValidateKey checks if the API key hash exists in Redis and returns the associated tenantID.
// Key format: auth:apikey:{hash} -> tenantID
func (r *RedisStore) ValidateKey(ctx context.Context, apiKeyHash string) (string, bool, error) {
	key := fmt.Sprintf("auth:apikey:%s", apiKeyHash)
	tenantID, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return tenantID, true, nil
}
