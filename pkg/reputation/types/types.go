package types

import (
	"time"
)

// Verification represents a stored reputation verification
type Verification struct {
	ID              string                 `json:"id"`
	UserID          string                 `json:"user_id"`
	TrustScore      float64                `json:"trust_score"`
	Confidence      float64                `json:"confidence"`
	Components      map[string]interface{} `json:"components"`
	ReceiptID       string                 `json:"receipt_id"`
	AlgorithmVersion string                `json:"algorithm_version"`
	CreatedAt       time.Time              `json:"created_at"`
	ExpiresAt       time.Time              `json:"expires_at"`
	LastRefreshedAt *time.Time             `json:"last_refreshed_at,omitempty"`
}

// PlatformConnection represents a connection to an external platform
type PlatformConnection struct {
	ID                 string                 `json:"id"`
	UserID             string                 `json:"user_id"`
	PlatformType       string                 `json:"platform_type"`
	PlatformUsername   string                 `json:"platform_username"`
	PlatformUserID     string                 `json:"platform_user_id,omitempty"`
	Verified           bool                   `json:"verified"`
	VerifiedAt         *time.Time             `json:"verified_at,omitempty"`
	VerificationMethod string                 `json:"verification_method,omitempty"`
	LastChecked        *time.Time             `json:"last_checked,omitempty"`
	LastScore          *float64               `json:"last_score,omitempty"`
	Metadata           map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt          time.Time              `json:"created_at"`
	UpdatedAt          time.Time              `json:"updated_at"`
}

// ReputationHistory represents a historical reputation snapshot
type ReputationHistory struct {
	ID               string    `json:"id"`
	UserID           string    `json:"user_id"`
	TrustScore       float64   `json:"trust_score"`
	Confidence       float64   `json:"confidence"`
	SnapshotDate     time.Time `json:"snapshot_date"`
	PlatformCount    int       `json:"platform_count"`
	AlgorithmVersion string    `json:"algorithm_version"`
	CreatedAt        time.Time `json:"created_at"`
}

// VerificationRequest represents an incoming verification request
type VerificationRequest struct {
	UserID    string            `json:"user_id"`
	Platforms []PlatformProfile `json:"platforms"`
	Weights   *ScoreWeights     `json:"weights,omitempty"`
}

// PlatformProfile represents a platform profile for verification
type PlatformProfile struct {
	PlatformType string                 `json:"platform_type"`
	Score        float64                `json:"score"`
	Weight       float64                `json:"weight"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// ScoreWeights represents weighting factors for reputation calculation
type ScoreWeights struct {
	Recency   float64 `json:"recency"`
	Volume    float64 `json:"volume"`
	Diversity float64 `json:"diversity"`
}

// DefaultWeights returns default score weights
func DefaultWeights() *ScoreWeights {
	return &ScoreWeights{
		Recency:   0.3,
		Volume:    0.3,
		Diversity: 0.4,
	}
}

// VerificationResponse represents the response from a verification request
type VerificationResponse struct {
	UserID       string                 `json:"user_id"`
	TrustScore   float64                `json:"trust_score"`
	Confidence   float64                `json:"confidence"`
	Components   map[string]interface{} `json:"components"`
	ReceiptID    string                 `json:"receipt_id"`
	ReceiptB64   string                 `json:"receipt_b64"`
	ExpiresAt    time.Time              `json:"expires_at"`
	VerifyURL    string                 `json:"verify_url"`
	BadgeURL     string                 `json:"badge_url"`
}

// BadgeConfig represents configuration for badge generation
type BadgeConfig struct {
	Style string `json:"style"` // "flat", "flat-square", "for-the-badge"
	Color string `json:"color"` // Custom color override
	Label string `json:"label"` // Custom label text
}

// Platform types (constants)
const (
	PlatformGitHub        = "github"
	PlatformLinkedIn      = "linkedin"
	PlatformTwitter       = "twitter"
	PlatformStackOverflow = "stackoverflow"
	PlatformMedium        = "medium"
	PlatformDevTo         = "devto"
	PlatformHashnode      = "hashnode"
)

// Verification method types
const (
	VerificationMethodOAuth     = "oauth"
	VerificationMethodAPIKey    = "api_key"
	VerificationMethodWebhook   = "webhook"
	VerificationMethodChallenge = "challenge"
)
