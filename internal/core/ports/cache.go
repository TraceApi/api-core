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
}
