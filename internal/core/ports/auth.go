/*
 * Copyright (c) 2025 Alessandro Faranda Gancio (dba TraceApi)
 *
 * This source code is licensed under the Business Source License 1.1.
 *
 * Change Date: 2027-11-28
 * Change License: AGPL-3.0
 */

package ports

import "context"

type AuthRepository interface {
	ValidateKey(ctx context.Context, apiKeyHash string) (tenantID string, valid bool, err error)
	GetTenantState(ctx context.Context, tenantID string) (state string, err error)
}
