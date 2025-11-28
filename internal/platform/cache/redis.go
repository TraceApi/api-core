/*
 * Copyright (c) 2025 Alessandro Faranda Gancio (dba TraceApi)
 *
 * This source code is licensed under the Business Source License 1.1.
 *
 * Change Date: 2027-11-21
 * Change License: AGPL-3.0
 */

package cache

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

var ErrCacheMiss = errors.New("key not found")

type RedisStore struct {
	client *redis.Client
}

func NewRedisStore(addr string) *RedisStore {
	// In a real app, we'd handle passwords via options
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: "", // No password set in docker-compose
		DB:       0,  // Use default DB
	})
	return &RedisStore{client: rdb}
}

// GetIdempotency checks if this operation hash exists.
// Returns the PassportID (string) if found, or error.
func (r *RedisStore) GetIdempotency(ctx context.Context, hash string) (string, error) {
	val, err := r.client.Get(ctx, "idempotency:"+hash).Result()
	if err == redis.Nil {
		return "", ErrCacheMiss
	}
	if err != nil {
		return "", err
	}
	return val, nil
}

// SetIdempotency saves the hash -> passportID mapping for 24 hours.
func (r *RedisStore) SetIdempotency(ctx context.Context, hash string, passportID string) error {
	// TTL = 24 hours. After that, we allow a duplicate to be created (or logic resets).
	return r.client.Set(ctx, "idempotency:"+hash, passportID, 24*time.Hour).Err()
}

// Get retrieves a value by key. Returns ErrCacheMiss if not found.
func (r *RedisStore) Get(ctx context.Context, key string) (string, error) {
	val, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", ErrCacheMiss
	}
	if err != nil {
		return "", err
	}
	return val, nil
}

// Set stores a value with a TTL (Time To Live).
func (r *RedisStore) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	return r.client.Set(ctx, key, value, ttl).Err()
}

func (r *RedisStore) Delete(ctx context.Context, key string) error {
	return r.client.Del(ctx, key).Err()
}
