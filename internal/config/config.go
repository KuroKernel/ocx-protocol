package config

import (
	"os"
	"strconv"
)

// Config holds all configuration for the OCX Protocol
type Config struct {
	// Database
	Database DatabaseConfig `json:"database"`
	
	// Payment Processing
	Stripe StripeConfig `json:"stripe"`
	USDC   USDCConfig   `json:"usdc"`
	
	// Identity Verification
	Jumio  JumioConfig  `json:"jumio"`
	Onfido OnfidoConfig `json:"onfido"`
	
	// Customer Support
	Zendesk  ZendeskConfig  `json:"zendesk"`
	Intercom IntercomConfig `json:"intercom"`
	
	// Verification
	Hardware HardwareConfig `json:"hardware"`
	
	// Legal
	Legal LegalConfig `json:"legal"`
	
	// Server
	Server ServerConfig `json:"server"`
}

type DatabaseConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
	DBName   string `json:"dbname"`
	SSLMode  string `json:"sslmode"`
}

type StripeConfig struct {
	SecretKey      string `json:"secret_key"`
	PublishableKey string `json:"publishable_key"`
	WebhookSecret  string `json:"webhook_secret"`
}

type USDCConfig struct {
	RPCURL        string `json:"rpc_url"`
	ContractAddr  string `json:"contract_address"`
	PrivateKey    string `json:"private_key"`
	GasPrice      int64  `json:"gas_price"`
}

type JumioConfig struct {
	APIKey    string `json:"api_key"`
	APISecret string `json:"api_secret"`
	BaseURL   string `json:"base_url"`
}

type OnfidoConfig struct {
	APIToken string `json:"api_token"`
	BaseURL  string `json:"base_url"`
}

type ZendeskConfig struct {
	Domain   string `json:"domain"`
	Email    string `json:"email"`
	APIToken string `json:"api_token"`
}

type IntercomConfig struct {
	AppID     string `json:"app_id"`
	APIToken  string `json:"api_token"`
	BaseURL   string `json:"base_url"`
}

type HardwareConfig struct {
	BenchmarkTimeout int `json:"benchmark_timeout"`
	MinScore         float64 `json:"min_score"`
}

type LegalConfig struct {
	TermsVersion    string `json:"terms_version"`
	PrivacyVersion  string `json:"privacy_version"`
	SLAVersion      string `json:"sla_version"`
}

type ServerConfig struct {
	Port        int    `json:"port"`
	Environment string `json:"environment"`
	LogLevel    string `json:"log_level"`
}

// LoadConfig loads configuration from environment variables
func LoadConfig() *Config {
	return &Config{
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnvInt("DB_PORT", 5432),
			User:     getEnv("DB_USER", "ocx_user"),
			Password: getEnv("DB_PASSWORD", "ocx_password"),
			DBName:   getEnv("DB_NAME", "ocx_protocol"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		},
		
		Stripe: StripeConfig{
			SecretKey:      getEnv("STRIPE_SECRET_KEY", ""),
			PublishableKey: getEnv("STRIPE_PUBLISHABLE_KEY", ""),
			WebhookSecret:  getEnv("STRIPE_WEBHOOK_SECRET", ""),
		},
		
		USDC: USDCConfig{
			RPCURL:       getEnv("USDC_RPC_URL", "https://mainnet.infura.io/v3/YOUR_PROJECT_ID"),
			ContractAddr: getEnv("USDC_CONTRACT_ADDRESS", "0xA0b86a33E6441b8c4C8C0C4C0C4C0C4C0C4C0C4C"),
			PrivateKey:   getEnv("USDC_PRIVATE_KEY", ""),
			GasPrice:     getEnvInt64("USDC_GAS_PRICE", 20000000000),
		},
		
		Jumio: JumioConfig{
			APIKey:    getEnv("JUMIO_API_KEY", ""),
			APISecret: getEnv("JUMIO_API_SECRET", ""),
			BaseURL:   getEnv("JUMIO_BASE_URL", "https://netverify.com/api/v4"),
		},
		
		Onfido: OnfidoConfig{
			APIToken: getEnv("ONFIDO_API_TOKEN", ""),
			BaseURL:  getEnv("ONFIDO_BASE_URL", "https://api.onfido.com/v3"),
		},
		
		Zendesk: ZendeskConfig{
			Domain:   getEnv("ZENDESK_DOMAIN", ""),
			Email:    getEnv("ZENDESK_EMAIL", ""),
			APIToken: getEnv("ZENDESK_API_TOKEN", ""),
		},
		
		Intercom: IntercomConfig{
			AppID:    getEnv("INTERCOM_APP_ID", ""),
			APIToken: getEnv("INTERCOM_API_TOKEN", ""),
			BaseURL:  getEnv("INTERCOM_BASE_URL", "https://api.intercom.io"),
		},
		
		Hardware: HardwareConfig{
			BenchmarkTimeout: getEnvInt("HARDWARE_BENCHMARK_TIMEOUT", 300),
			MinScore:         getEnvFloat64("HARDWARE_MIN_SCORE", 0.8),
		},
		
		Legal: LegalConfig{
			TermsVersion:   getEnv("LEGAL_TERMS_VERSION", "1.0.0"),
			PrivacyVersion: getEnv("LEGAL_PRIVACY_VERSION", "1.0.0"),
			SLAVersion:     getEnv("LEGAL_SLA_VERSION", "1.0.0"),
		},
		
		Server: ServerConfig{
			Port:        getEnvInt("SERVER_PORT", 8080),
			Environment: getEnv("ENVIRONMENT", "development"),
			LogLevel:    getEnv("LOG_LEVEL", "info"),
		},
	}
}

// Helper functions
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvInt64(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.ParseInt(value, 10, 64); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvFloat64(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if floatValue, err := strconv.ParseFloat(value, 64); err == nil {
			return floatValue
		}
	}
	return defaultValue
}
