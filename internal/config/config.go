package config

import (
	"os"
	"strings"
)

type Config struct {
	Environment string
	Port        string
	DatabaseURL string
	RedisAddr   string
	LogLevel    string
	JWTSecret   string

	// S3 / Minio
	S3Endpoint  string
	S3Region    string
	S3AccessKey string
	S3SecretKey string
	S3Bucket    string
}

// Load returns the application configuration from environment variables
func Load() *Config {
	return &Config{
		Environment: getEnv("APP_ENV", "development"),
		Port:        getEnv("PORT", "8080"),
		DatabaseURL: getEnv("DATABASE_URL", "postgres://trace_user:trace_password@localhost:5432/trace_core?sslmode=disable"),
		RedisAddr:   getEnv("REDIS_ADDR", "localhost:6379"),
		LogLevel:    getEnv("LOG_LEVEL", "info"),
		JWTSecret:   getEnv("JWT_SECRET", "super-secret-dev-key-do-not-use-in-prod"),

		S3Endpoint:  getEnv("S3_ENDPOINT", "http://localhost:9000"),
		S3Region:    getEnv("S3_REGION", "us-east-1"),
		S3AccessKey: getEnv("S3_ACCESS_KEY", "minio_admin"),
		S3SecretKey: getEnv("S3_SECRET_KEY", "minio_password"),
		S3Bucket:    getEnv("S3_BUCKET", "passports"),
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func (c *Config) IsProduction() bool {
	return strings.ToLower(c.Environment) == "production"
}
