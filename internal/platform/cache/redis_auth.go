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
