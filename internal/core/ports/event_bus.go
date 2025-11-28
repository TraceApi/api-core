package ports

import "context"

type EventBus interface {
	Publish(ctx context.Context, channel string, event interface{}) error
}
