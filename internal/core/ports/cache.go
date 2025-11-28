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
	"time"
)

type CacheRepository interface {
	GetIdempotency(ctx context.Context, hash string) (string, error)
	SetIdempotency(ctx context.Context, hash string, passportID string) error
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value string, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
}
