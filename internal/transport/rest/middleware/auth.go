package middleware

import (
"context"
"fmt"
"log/slog"
"net/http"
"strings"

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

// GetManufacturerID retrieves the ID from context
func GetManufacturerID(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(ManufacturerIDKey).(string)
	return id, ok
}
