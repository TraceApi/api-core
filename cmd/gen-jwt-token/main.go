/*
 * Copyright (c) 2025 Alessandro Faranda Gancio (dba TraceApi)
 *
 * This source code is licensed under the Business Source License 1.1.
 *
 * Change Date: 2027-11-28
 * Change License: AGPL-3.0
 */

package main

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func main() {
	// Must match the secret in internal/config/config.go
	secret := []byte("super-secret-dev-key-do-not-use-in-prod")

	claims := jwt.MapClaims{
		"sub":             "manufacturer-001",
		"manufacturer_id": "manufacturer-001",
		"exp":             time.Now().Add(24 * time.Hour).Unix(),
		"iat":             time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(secret)
	if err != nil {
		panic(err)
	}

	fmt.Println("Generated JWT Token:")
	fmt.Println(tokenString)
	fmt.Println("\nCurl Command:")
	fmt.Printf("curl -v -X POST http://localhost:8080/passports?category=BATTERY_INDUSTRIAL \\\n  -H \"Authorization: Bearer %s\" \\\n  -d '{}'\n", tokenString)
}
