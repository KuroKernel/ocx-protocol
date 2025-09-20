package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port            string
	DatabaseURL     string
	DatabaseType    string // "sqlite" or "postgres"
	MetricsEnabled  bool
	LogLevel        string
	KeystoreDir     string
	RateLimitRPS    int
	MaxReceiptSize  int64
	MaxBodyBytes    int64
}

func Load() *Config {
	cfg := &Config{
		Port:           getEnv("PORT", "8080"),
		DatabaseType:   getEnv("DB_TYPE", "sqlite"),
		DatabaseURL:    getEnv("DATABASE_URL", "ocx.db"),
		MetricsEnabled: getEnvBool("METRICS_ENABLED", true),
		LogLevel:       getEnv("LOG_LEVEL", "info"),
		KeystoreDir:    getEnv("KEYSTORE_DIR", "./keys"),
		RateLimitRPS:   getEnvInt("RATE_LIMIT_RPS", 100),
		MaxReceiptSize: getEnvInt64("MAX_RECEIPT_SIZE", 1024*1024), // 1MB
		MaxBodyBytes:   getEnvInt64("OCX_MAX_BODY_BYTES", 1024*1024), // 1MB
	}

	// Override for production
	if cfg.DatabaseType == "postgres" && cfg.DatabaseURL == "ocx.db" {
		cfg.DatabaseURL = getEnv("POSTGRES_URL", "postgres://ocx:password@localhost/ocx?sslmode=disable")
	}

	return cfg
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseBool(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}

func getEnvInt64(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseInt(value, 10, 64); err == nil {
			return parsed
		}
	}
	return defaultValue
}
