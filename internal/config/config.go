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
}

// Load returns the application configuration from environment variables
func Load() *Config {
	return &Config{
		Environment: getEnv("APP_ENV", "development"),
		Port:        getEnv("PORT", "8080"),
		DatabaseURL: getEnv("DATABASE_URL", "postgres://trace_user:trace_password@localhost:5432/trace_core?sslmode=disable"),
		RedisAddr:   getEnv("REDIS_ADDR", "localhost:6379"),
		LogLevel:    getEnv("LOG_LEVEL", "info"),
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
