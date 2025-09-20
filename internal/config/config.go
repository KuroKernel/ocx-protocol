package config

import (
    "os"
    "strconv"
    "time"
)

type Config struct {
    Port            string
    DatabaseURL     string
    MaxCycles       uint64
    RequestTimeout  time.Duration
    RateLimit       int
    EnableMetrics   bool
    LogLevel        string
}

func LoadConfig() *Config {
    return &Config{
        Port:            getEnv("OCX_PORT", "8080"),
        DatabaseURL:     getEnv("OCX_DATABASE_URL", ":memory:"),
        MaxCycles:       getEnvUint64("OCX_MAX_CYCLES", 1000000),
        RequestTimeout:  getEnvDuration("OCX_REQUEST_TIMEOUT", "30s"),
        RateLimit:       getEnvInt("OCX_RATE_LIMIT", 1000),
        EnableMetrics:   getEnvBool("OCX_ENABLE_METRICS", true),
        LogLevel:        getEnv("OCX_LOG_LEVEL", "info"),
    }
}

func getEnv(key, defaultValue string) string {
    if value := os.Getenv(key); value != "" {
        return value
    }
    return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
    if value := os.Getenv(key); value != "" {
        if i, err := strconv.Atoi(value); err == nil {
            return i
        }
    }
    return defaultValue
}

func getEnvUint64(key string, defaultValue uint64) uint64 {
    if value := os.Getenv(key); value != "" {
        if i, err := strconv.ParseUint(value, 10, 64); err == nil {
            return i
        }
    }
    return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
    if value := os.Getenv(key); value != "" {
        if b, err := strconv.ParseBool(value); err == nil {
            return b
        }
    }
    return defaultValue
}

func getEnvDuration(key string, defaultValue string) time.Duration {
    if value := os.Getenv(key); value != "" {
        if d, err := time.ParseDuration(value); err == nil {
            return d
        }
    }
    if d, err := time.ParseDuration(defaultValue); err == nil {
        return d
    }
    return time.Second * 30
}
