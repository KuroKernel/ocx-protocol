package config

import (
	"os"
)

type Config struct {
	Port      string // ":8081"
	APIKey    string
	DisableDB bool
	DSN       string
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func Load() Config {
	port := getenv("OCX_PORT", "8080")
	// Remove leading colon if present (server will add it)
	if port != "" && port[0] == ':' {
		port = port[1:]
	}

	return Config{
		Port:      port,
		APIKey:    getenv("OCX_API_KEY", ""),
		DisableDB: getenv("OCX_DISABLE_DB", "") == "true",
		DSN:       getenv("OCX_DB_DSN", ""),
	}
}
