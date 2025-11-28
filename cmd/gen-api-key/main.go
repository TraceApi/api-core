package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
)

func main() {
	tenantID := flag.String("tenant", "manufacturer-001", "The Tenant ID to associate with this key")
	flag.Parse()

	// 1. Generate a random 32-byte hex string (64 chars) as the API Key
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		fmt.Println("Error generating random bytes:", err)
		os.Exit(1)
	}
	apiKey := hex.EncodeToString(bytes)

	// 2. Calculate SHA-256 Hash
	hash := sha256.Sum256([]byte(apiKey))
	apiKeyHash := hex.EncodeToString(hash[:])

	// 3. Output
	fmt.Println("=== New API Key Generated ===")
	fmt.Printf("Raw API Key (Client Use): %s\n", apiKey)
	fmt.Printf("Tenant ID:                %s\n", *tenantID)
	fmt.Println("\n=== Redis Setup Command ===")
	fmt.Println("Run this command in your Redis instance to register the key:")
	fmt.Printf("SET auth:apikey:%s \"%s\"\n", apiKeyHash, *tenantID)
	fmt.Println("\n=== Curl Example ===")
	fmt.Printf("curl -v -H \"Authorization: Bearer %s\" http://localhost:8080/health\n", apiKey)
}
