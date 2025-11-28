package bus

import (
	"context"
	"encoding/json"

	"github.com/redis/go-redis/v9"
)

type RedisEventBus struct {
	client *redis.Client
}

func NewRedisEventBus(addr string) *RedisEventBus {
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: "", // No password set in docker-compose
		DB:       0,  // Use default DB
	})
	return &RedisEventBus{client: rdb}
}

func (b *RedisEventBus) Publish(ctx context.Context, channel string, event interface{}) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}
	return b.client.Publish(ctx, channel, payload).Err()
}
