package ports

import "context"

type AuthRepository interface {
	ValidateKey(ctx context.Context, apiKeyHash string) (tenantID string, valid bool, err error)
}
