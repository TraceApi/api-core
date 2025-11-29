/*
 * Copyright (c) 2025 Alessandro Faranda Gancio (dba TraceApi)
 *
 * This source code is licensed under the Business Source License 1.1.
 *
 * Change Date: 2027-11-28
 * Change License: AGPL-3.0
 */

package middleware

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/TraceApi/api-core/internal/core/ports"
	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const (
	ManufacturerIDKey contextKey = "manufacturer_id"
)

func HybridAuthMiddleware(jwtSecret string, authRepo ports.AuthRepository, log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "missing authorization header", http.StatusUnauthorized)
				return
			}

			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				http.Error(w, "invalid authorization format", http.StatusUnauthorized)
				return
			}
			tokenString := parts[1]

			var tenantID string

			// ---------------------------------------------------------
			// PHASE 1: IDENTIFICATION
			// ---------------------------------------------------------

			if strings.HasPrefix(tokenString, "traceapi_") {
				// --- STRATEGY A: API KEY ---
				hash := sha256.Sum256([]byte(tokenString))
				apiKeyHash := hex.EncodeToString(hash[:])

				id, valid, err := authRepo.ValidateKey(r.Context(), apiKeyHash)
				if err != nil {
					log.Error("auth validation error", "error", err)
					http.Error(w, "internal server error", http.StatusInternalServerError)
					return
				}
				if !valid {
					http.Error(w, "invalid api key", http.StatusUnauthorized)
					return
				}
				tenantID = id

			} else {
				// --- STRATEGY B: JWT ---
				token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
					if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
						return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
					}
					return []byte(jwtSecret), nil
				})

				if err != nil || !token.Valid {
					http.Error(w, "invalid or expired token", http.StatusUnauthorized)
					return
				}

				claims, ok := token.Claims.(jwt.MapClaims)
				if !ok {
					http.Error(w, "invalid token claims", http.StatusUnauthorized)
					return
				}

				sub, err := claims.GetSubject()
				if err != nil || sub == "" {
					if mfgID, ok := claims["manufacturer_id"].(string); ok {
						sub = mfgID
					} else {
						log.Warn("token missing subject claim")
						http.Error(w, "token missing subject", http.StatusUnauthorized)
						return
					}
				}
				tenantID = sub
			}

			// ---------------------------------------------------------
			// PHASE 2: AUTHORIZATION
			// ---------------------------------------------------------
			// This is the "Circuit Breaker". It applies to BOTH API Keys and JWTs.

			state, err := authRepo.GetTenantState(r.Context(), tenantID)
			if err != nil {
				// Fail CLOSED. If Redis is down, we can't verify quota.
				log.Error("failed to check tenant state", "error", err)
				http.Error(w, "system error", http.StatusInternalServerError)
				return
			}

			if state == "BLOCKED" {
				// Quota exceeded or Bill unpaid
				http.Error(w, "quota exceeded or payment required", 402)
				return
			}

			// ---------------------------------------------------------
			// PHASE 3: EXECUTION
			// ---------------------------------------------------------
			ctx := context.WithValue(r.Context(), ManufacturerIDKey, tenantID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
