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

// AuthMiddleware handles JWT validation
func AuthMiddleware(secret string, log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "missing authorization header", http.StatusUnauthorized)
				return
			}

			// Expect format: "Bearer <token>"
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				http.Error(w, "invalid authorization format", http.StatusUnauthorized)
				return
			}

			tokenString := parts[1]

			// Parse and validate the token
			token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
				// Validate the signing method (IMPORTANT security step)
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
				}
				// In a real OIDC setup, you would fetch the Public Key (JWKS) here based on the 'kid' header
				return []byte(secret), nil
			})

			if err != nil || !token.Valid {
				log.Warn("invalid token", "error", err)
				http.Error(w, "invalid or expired token", http.StatusUnauthorized)
				return
			}

			// Extract Claims
			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				http.Error(w, "invalid token claims", http.StatusUnauthorized)
				return
			}

			// Get the Subject (sub) which usually holds the User ID or Manufacturer ID
			sub, err := claims.GetSubject()
			if err != nil || sub == "" {
				// Fallback: Try custom claim if 'sub' is missing (depends on your IdP)
				if mfgID, ok := claims["manufacturer_id"].(string); ok {
					sub = mfgID
				} else {
					log.Warn("token missing subject claim")
					http.Error(w, "token missing subject", http.StatusUnauthorized)
					return
				}
			}

			// Inject into Context
			ctx := context.WithValue(r.Context(), ManufacturerIDKey, sub)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// APIKeyAuthMiddleware handles API Key validation
func APIKeyAuthMiddleware(authRepo ports.AuthRepository, log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "missing authorization header", http.StatusUnauthorized)
				return
			}

			// Expect format: "Bearer <token>"
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				http.Error(w, "invalid authorization format", http.StatusUnauthorized)
				return
			}

			tokenString := parts[1]

			// Hash the token (SHA-256)
			hash := sha256.Sum256([]byte(tokenString))
			apiKeyHash := hex.EncodeToString(hash[:])

			// Call ValidateKey
			tenantID, valid, err := authRepo.ValidateKey(r.Context(), apiKeyHash)
			if err != nil {
				log.Error("auth validation error", "error", err)
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}

			if !valid {
				http.Error(w, "invalid api key", http.StatusUnauthorized)
				return
			}

			// Inject tenantID into Context
			ctx := context.WithValue(r.Context(), ManufacturerIDKey, tenantID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// HybridAuthMiddleware handles both JWT and API Key validation
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

			// 1. Try JWT
			token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
				}
				return []byte(jwtSecret), nil
			})

			if err == nil && token.Valid {
				// Valid JWT
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

				ctx := context.WithValue(r.Context(), ManufacturerIDKey, sub)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			// 2. Try API Key
			hash := sha256.Sum256([]byte(tokenString))
			apiKeyHash := hex.EncodeToString(hash[:])
			tenantID, valid, err := authRepo.ValidateKey(r.Context(), apiKeyHash)
			if err != nil {
				log.Error("auth validation error", "error", err)
				http.Error(w, "internal server error", http.StatusInternalServerError)
				return
			}

			if valid {
				ctx := context.WithValue(r.Context(), ManufacturerIDKey, tenantID)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			// Both failed
			http.Error(w, "invalid or expired token", http.StatusUnauthorized)
		})
	}
}

// GetManufacturerID retrieves the ID from context
func GetManufacturerID(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(ManufacturerIDKey).(string)
	return id, ok
}
