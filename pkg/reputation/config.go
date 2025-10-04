package reputation

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
)

// WASMConfig holds configuration for the TrustScore WASM module
type WASMConfig struct {
	// ArtifactPath is the path to the compiled WASM module
	ArtifactPath string

	// ArtifactHash is the SHA256 hash of the WASM module
	ArtifactHash [32]byte

	// Enabled indicates if reputation verification is enabled
	Enabled bool
}

var defaultConfig *WASMConfig

// GetWASMConfig returns the default WASM configuration
func GetWASMConfig() *WASMConfig {
	if defaultConfig != nil {
		return defaultConfig
	}

	// Load configuration
	artifactPath := os.Getenv("TRUSTSCORE_WASM_PATH")
	if artifactPath == "" {
		artifactPath = "./artifacts/trustscore.wasm"
	}

	// Check if WASM module exists
	if _, err := os.Stat(artifactPath); os.IsNotExist(err) {
		return &WASMConfig{
			ArtifactPath: artifactPath,
			Enabled:      false,
		}
	}

	// Calculate artifact hash
	hash, err := calculateFileHash(artifactPath)
	if err != nil {
		return &WASMConfig{
			ArtifactPath: artifactPath,
			Enabled:      false,
		}
	}

	defaultConfig = &WASMConfig{
		ArtifactPath: artifactPath,
		ArtifactHash: hash,
		Enabled:      true,
	}

	return defaultConfig
}

// GetTrustScoreWASMHash returns the hash of the TrustScore WASM module
func GetTrustScoreWASMHash() [32]byte {
	config := GetWASMConfig()
	return config.ArtifactHash
}

// calculateFileHash computes SHA256 hash of a file
func calculateFileHash(path string) ([32]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return [32]byte{}, err
	}
	return sha256.Sum256(data), nil
}

// GetArtifactHashHex returns the artifact hash as a hex string
func GetArtifactHashHex() string {
	config := GetWASMConfig()
	return hex.EncodeToString(config.ArtifactHash[:])
}

// IsEnabled checks if reputation verification is enabled
func IsEnabled() bool {
	return GetWASMConfig().Enabled
}
