package artifacts

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// ArtifactConfig represents the configuration for the artifact resolution system
type ArtifactConfig struct {
	Cache struct {
		MemorySizeMB  int    `yaml:"memory_size_mb"`
		DiskSizeGB    int    `yaml:"disk_size_gb"`
		TTLMinutes    int    `yaml:"ttl_minutes"`
		BaseDirectory string `yaml:"base_directory"`
		ShardCount    int    `yaml:"shard_count"`
	} `yaml:"cache"`

	Remote struct {
		Sources                []ArtifactSource `yaml:"sources"`
		TimeoutSeconds         int              `yaml:"timeout_seconds"`
		MaxRetries             int              `yaml:"max_retries"`
		BackoffSeconds         int              `yaml:"backoff_seconds"`
		MaxConcurrentDownloads int              `yaml:"max_concurrent_downloads"`
	} `yaml:"remote"`

	Security struct {
		VerifySignatures bool   `yaml:"verify_signatures"`
		PublicKeyPath    string `yaml:"public_key_path"`
		TLSCertPath      string `yaml:"tls_cert_path"`
	} `yaml:"security"`

	Performance struct {
		EnableMetrics   bool `yaml:"enable_metrics"`
		MetricsPort     int  `yaml:"metrics_port"`
		EnableProfiling bool `yaml:"enable_profiling"`
		ProfilingPort   int  `yaml:"profiling_port"`
	} `yaml:"performance"`
}

// DefaultArtifactConfig returns a default configuration
func DefaultArtifactConfig() *ArtifactConfig {
	return &ArtifactConfig{
		Cache: struct {
			MemorySizeMB  int    `yaml:"memory_size_mb"`
			DiskSizeGB    int    `yaml:"disk_size_gb"`
			TTLMinutes    int    `yaml:"ttl_minutes"`
			BaseDirectory string `yaml:"base_directory"`
			ShardCount    int    `yaml:"shard_count"`
		}{
			MemorySizeMB:  512, // 512MB memory cache
			DiskSizeGB:    10,  // 10GB disk cache
			TTLMinutes:    60,  // 1 hour TTL
			BaseDirectory: "/var/cache/ocx/artifacts",
			ShardCount:    256, // 256 shards for good distribution
		},
		Remote: struct {
			Sources                []ArtifactSource `yaml:"sources"`
			TimeoutSeconds         int              `yaml:"timeout_seconds"`
			MaxRetries             int              `yaml:"max_retries"`
			BackoffSeconds         int              `yaml:"backoff_seconds"`
			MaxConcurrentDownloads int              `yaml:"max_concurrent_downloads"`
		}{
			Sources: []ArtifactSource{
				{
					URL:          "https://artifacts.ocx.local",
					Priority:     1,
					AuthRequired: false,
					Timeout:      30 * time.Second,
					RateLimit:    100, // 100 requests per second
				},
			},
			TimeoutSeconds:         30,
			MaxRetries:             3,
			BackoffSeconds:         1,
			MaxConcurrentDownloads: 10,
		},
		Security: struct {
			VerifySignatures bool   `yaml:"verify_signatures"`
			PublicKeyPath    string `yaml:"public_key_path"`
			TLSCertPath      string `yaml:"tls_cert_path"`
		}{
			VerifySignatures: true,
			PublicKeyPath:    "/etc/ocx/keys/artifact-verification.pub",
			TLSCertPath:      "/etc/ocx/certs/tls.crt",
		},
		Performance: struct {
			EnableMetrics   bool `yaml:"enable_metrics"`
			MetricsPort     int  `yaml:"metrics_port"`
			EnableProfiling bool `yaml:"enable_profiling"`
			ProfilingPort   int  `yaml:"profiling_port"`
		}{
			EnableMetrics:   true,
			MetricsPort:     9090,
			EnableProfiling: false,
			ProfilingPort:   6060,
		},
	}
}

// LoadArtifactConfig loads configuration from a YAML file
func LoadArtifactConfig(configPath string) (*ArtifactConfig, error) {
	if configPath == "" {
		return DefaultArtifactConfig(), nil
	}

	// Check if file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file not found: %s", configPath)
	}

	// Read file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse YAML
	var config ArtifactConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &config, nil
}

// SaveArtifactConfig saves configuration to a YAML file
func SaveArtifactConfig(config *ArtifactConfig, configPath string) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Marshal to YAML
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write to file
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// Validate validates the configuration
func (ac *ArtifactConfig) Validate() error {
	// Validate cache configuration
	if ac.Cache.MemorySizeMB <= 0 {
		return fmt.Errorf("memory_size_mb must be positive, got %d", ac.Cache.MemorySizeMB)
	}
	if ac.Cache.MemorySizeMB > 8192 {
		return fmt.Errorf("memory_size_mb too large, maximum 8192MB, got %d", ac.Cache.MemorySizeMB)
	}

	if ac.Cache.DiskSizeGB <= 0 {
		return fmt.Errorf("disk_size_gb must be positive, got %d", ac.Cache.DiskSizeGB)
	}
	if ac.Cache.DiskSizeGB > 1000 {
		return fmt.Errorf("disk_size_gb too large, maximum 1000GB, got %d", ac.Cache.DiskSizeGB)
	}

	if ac.Cache.TTLMinutes <= 0 {
		return fmt.Errorf("ttl_minutes must be positive, got %d", ac.Cache.TTLMinutes)
	}

	if ac.Cache.BaseDirectory == "" {
		return fmt.Errorf("base_directory cannot be empty")
	}

	if ac.Cache.ShardCount <= 0 {
		return fmt.Errorf("shard_count must be positive, got %d", ac.Cache.ShardCount)
	}
	if ac.Cache.ShardCount > 1024 {
		return fmt.Errorf("shard_count too large, maximum 1024, got %d", ac.Cache.ShardCount)
	}

	// Validate remote configuration
	if len(ac.Remote.Sources) == 0 {
		return fmt.Errorf("at least one remote source must be configured")
	}

	for i, source := range ac.Remote.Sources {
		if source.URL == "" {
			return fmt.Errorf("source %d: URL cannot be empty", i)
		}
		if source.Priority < 0 {
			return fmt.Errorf("source %d: priority must be non-negative, got %d", i, source.Priority)
		}
		if source.Timeout <= 0 {
			return fmt.Errorf("source %d: timeout must be positive, got %v", i, source.Timeout)
		}
	}

	if ac.Remote.TimeoutSeconds <= 0 {
		return fmt.Errorf("timeout_seconds must be positive, got %d", ac.Remote.TimeoutSeconds)
	}
	if ac.Remote.TimeoutSeconds > 300 {
		return fmt.Errorf("timeout_seconds too large, maximum 300s, got %d", ac.Remote.TimeoutSeconds)
	}

	if ac.Remote.MaxRetries < 0 {
		return fmt.Errorf("max_retries must be non-negative, got %d", ac.Remote.MaxRetries)
	}
	if ac.Remote.MaxRetries > 10 {
		return fmt.Errorf("max_retries too large, maximum 10, got %d", ac.Remote.MaxRetries)
	}

	if ac.Remote.BackoffSeconds <= 0 {
		return fmt.Errorf("backoff_seconds must be positive, got %d", ac.Remote.BackoffSeconds)
	}

	if ac.Remote.MaxConcurrentDownloads <= 0 {
		return fmt.Errorf("max_concurrent_downloads must be positive, got %d", ac.Remote.MaxConcurrentDownloads)
	}
	if ac.Remote.MaxConcurrentDownloads > 100 {
		return fmt.Errorf("max_concurrent_downloads too large, maximum 100, got %d", ac.Remote.MaxConcurrentDownloads)
	}

	// Validate security configuration
	if ac.Security.VerifySignatures {
		if ac.Security.PublicKeyPath == "" {
			return fmt.Errorf("public_key_path is required when signature verification is enabled")
		}
	}

	// Validate performance configuration
	if ac.Performance.MetricsPort <= 0 || ac.Performance.MetricsPort > 65535 {
		return fmt.Errorf("metrics_port must be between 1 and 65535, got %d", ac.Performance.MetricsPort)
	}

	if ac.Performance.EnableProfiling {
		if ac.Performance.ProfilingPort <= 0 || ac.Performance.ProfilingPort > 65535 {
			return fmt.Errorf("profiling_port must be between 1 and 65535, got %d", ac.Performance.ProfilingPort)
		}
	}

	return nil
}

// GetCacheDirectory returns the full path to the cache directory
func (ac *ArtifactConfig) GetCacheDirectory() string {
	return ac.Cache.BaseDirectory
}

// GetMemoryCacheSize returns the memory cache size in bytes
func (ac *ArtifactConfig) GetMemoryCacheSize() int64 {
	return int64(ac.Cache.MemorySizeMB) * 1024 * 1024
}

// GetDiskCacheSize returns the disk cache size in bytes
func (ac *ArtifactConfig) GetDiskCacheSize() int64 {
	return int64(ac.Cache.DiskSizeGB) * 1024 * 1024 * 1024
}

// GetTTL returns the TTL as a duration
func (ac *ArtifactConfig) GetTTL() time.Duration {
	return time.Duration(ac.Cache.TTLMinutes) * time.Minute
}

// GetTimeout returns the timeout as a duration
func (ac *ArtifactConfig) GetTimeout() time.Duration {
	return time.Duration(ac.Remote.TimeoutSeconds) * time.Second
}

// GetBackoffDelay returns the backoff delay as a duration
func (ac *ArtifactConfig) GetBackoffDelay() time.Duration {
	return time.Duration(ac.Remote.BackoffSeconds) * time.Second
}

// IsSignatureVerificationEnabled returns whether signature verification is enabled
func (ac *ArtifactConfig) IsSignatureVerificationEnabled() bool {
	return ac.Security.VerifySignatures
}

// GetPublicKeyPath returns the path to the public key file
func (ac *ArtifactConfig) GetPublicKeyPath() string {
	return ac.Security.PublicKeyPath
}

// GetTLSCertPath returns the path to the TLS certificate file
func (ac *ArtifactConfig) GetTLSCertPath() string {
	return ac.Security.TLSCertPath
}

// IsMetricsEnabled returns whether metrics are enabled
func (ac *ArtifactConfig) IsMetricsEnabled() bool {
	return ac.Performance.EnableMetrics
}

// GetMetricsPort returns the metrics port
func (ac *ArtifactConfig) GetMetricsPort() int {
	return ac.Performance.MetricsPort
}

// IsProfilingEnabled returns whether profiling is enabled
func (ac *ArtifactConfig) IsProfilingEnabled() bool {
	return ac.Performance.EnableProfiling
}

// GetProfilingPort returns the profiling port
func (ac *ArtifactConfig) GetProfilingPort() int {
	return ac.Performance.ProfilingPort
}
