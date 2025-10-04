# OCX Protocol + TrustScore Integration Specification

## Executive Summary

**Integration Objective**: Add TrustScore reputation verification as a second application layer on top of existing OCX deterministic execution infrastructure, demonstrating protocol versatility while creating a consumer revenue stream and de-risking enterprise L2 sales through proof of scale.

**Zero Capital Constraint**: 100% organic growth, no paid acquisition, maximum code reuse from existing OCX infrastructure.

**Timeline**: 4-6 weeks for MVP, 12 weeks for production-ready

**Revenue Model**:
- **B2C**: Freemium verification API ($0 → $9/mo → $29/mo)
- **B2B**: Enterprise L2 fraud proofs (existing, $50k-$500k ARR)

---

## Current OCX Infrastructure Analysis

### Existing Capabilities (Ready to Reuse)

#### 1. Deterministic Execution Engine ✅
**Location**: `pkg/deterministicvm/`
- **Capability**: Executes WASM/ELF/scripts deterministically
- **Performance**: <1ms overhead, 100% consistency
- **TrustScore Use**: Run reputation aggregation algorithms deterministically
- **Zero New Code**: Existing VM handles all execution

#### 2. Cryptographic Receipt System ✅
**Location**: `pkg/receipt/`
- **Capability**: Ed25519-signed CBOR receipts
- **Performance**: ~600µs generation, ~670µs verification
- **TrustScore Use**: Sign reputation scores cryptographically
- **Zero New Code**: Existing receipt infrastructure works as-is

#### 3. HTTP API Server ✅
**Location**: `cmd/server/main.go` (2,329 LOC)
- **Capability**: Production-grade REST API with middleware
- **Features**: Auth, rate limiting, idempotency, metrics
- **TrustScore Use**: Add `/api/v1/reputation/*` endpoints
- **Minimal New Code**: ~300 LOC for new handlers

#### 4. Database Infrastructure ✅
**Location**: `pkg/database/`
- **Capability**: PostgreSQL/SQLite with connection pooling
- **Features**: Health checks, migrations, backup/recovery
- **TrustScore Use**: Store reputation data, verification history
- **New Code**: Only schema additions (1 migration file)

#### 5. Verification System ✅
**Location**: `pkg/verify/` + `libocx-verify/`
- **Capability**: Dual verification (Go + Rust)
- **Performance**: Sub-millisecond verification
- **TrustScore Use**: Verify reputation receipts independently
- **Zero New Code**: Existing verifiers work for reputation receipts

#### 6. Enterprise Features ✅
**Location**: Multiple packages
- **Monitoring**: Prometheus + Grafana dashboards
- **Backup**: Automated backup/disaster recovery
- **Compliance**: SOX, GDPR, HIPAA audit trails
- **Scaling**: Load balancing, clustering, caching
- **TrustScore Use**: All features apply to reputation data
- **Zero New Code**: Infrastructure already exists

---

## Integration Architecture

### System Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                    OCX PROTOCOL CORE                            │
│  (Deterministic VM + Receipt System + API Server)               │
│                                                                  │
│  ┌────────────────┐      ┌──────────────────┐                  │
│  │   D-MVM Engine │◄────►│ Receipt Generator│                  │
│  │   (Existing)   │      │   (Existing)     │                  │
│  └────────┬───────┘      └────────┬─────────┘                  │
│           │                       │                             │
│           ▼                       ▼                             │
│  ┌───────────────────────────────────────────────┐             │
│  │        APPLICATION LAYER ROUTER                │             │
│  │                                                 │             │
│  │  ┌──────────────┐  ┌──────────────────────┐  │             │
│  │  │  L2 Fraud    │  │  Reputation         │  │             │
│  │  │  Proofs      │  │  Verification       │  │             │
│  │  │  (Existing)  │  │  (NEW - TrustScore) │  │             │
│  │  └──────────────┘  └──────────────────────┘  │             │
│  └───────────────────────────────────────────────┘             │
└─────────────────────────────────────────────────────────────────┘
```

### Integration Points

#### 1. WASM Reputation Module (NEW)
**Location**: `modules/reputation/trustscore.wasm`
**Size**: ~50KB compiled WASM
**Language**: Rust (compile to WASM)
**Purpose**: Deterministic reputation aggregation

**Inputs**:
```json
{
  "user_id": "alice@example.com",
  "platforms": [
    {"type": "github", "username": "alice"},
    {"type": "linkedin", "profile_url": "..."},
    {"type": "twitter", "handle": "@alice"}
  ],
  "weights": {
    "recency": 0.3,
    "volume": 0.3,
    "diversity": 0.4
  }
}
```

**Outputs**:
```json
{
  "trust_score": 87.5,
  "confidence": 0.92,
  "components": {
    "github": {"score": 95, "weight": 0.4},
    "linkedin": {"score": 82, "weight": 0.3},
    "twitter": {"score": 85, "weight": 0.3}
  },
  "timestamp": 1696348800,
  "deterministic_hash": "sha256:abc123..."
}
```

**Execution**: `deterministicvm.ExecuteArtifact(wasmHash, inputJSON)`

#### 2. API Endpoints (NEW)
**Location**: `cmd/server/main.go` - Add reputation handlers

```go
// New handlers to add
func (s *Server) handleReputationVerify(w http.ResponseWriter, r *http.Request)
func (s *Server) handleReputationHistory(w http.ResponseWriter, r *http.Request)
func (s *Server) handleReputationBadge(w http.ResponseWriter, r *http.Request)
```

**Endpoints**:
```
POST   /api/v1/reputation/verify
GET    /api/v1/reputation/history/:user_id
GET    /api/v1/reputation/badge/:user_id
POST   /api/v1/reputation/refresh
GET    /api/v1/reputation/stats
```

#### 3. Database Schema (NEW)
**Location**: `database/migrations/002_trustscore.sql`

```sql
-- Reputation verifications table
CREATE TABLE reputation_verifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    trust_score DECIMAL(5,2) NOT NULL,
    confidence DECIMAL(4,3) NOT NULL,
    components JSONB NOT NULL,
    receipt_id VARCHAR(255) NOT NULL REFERENCES receipts(id),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    INDEX idx_user_id (user_id),
    INDEX idx_created_at (created_at)
);

-- Platform connections table
CREATE TABLE platform_connections (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    platform_type VARCHAR(50) NOT NULL,
    platform_username VARCHAR(255) NOT NULL,
    verified BOOLEAN DEFAULT FALSE,
    verified_at TIMESTAMP WITH TIME ZONE,
    last_checked TIMESTAMP WITH TIME ZONE,
    INDEX idx_user_platform (user_id, platform_type)
);

-- Reputation history table (for trend analysis)
CREATE TABLE reputation_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    trust_score DECIMAL(5,2) NOT NULL,
    snapshot_date DATE NOT NULL,
    INDEX idx_user_date (user_id, snapshot_date)
);
```

#### 4. Platform Adapters (NEW)
**Location**: `pkg/reputation/adapters/`

Each adapter implements standard interface:
```go
type PlatformAdapter interface {
    FetchProfile(ctx context.Context, username string) (*Profile, error)
    CalculateScore(profile *Profile) (float64, error)
    ValidateConnection(userID, username string) error
}

// Implementations
- github_adapter.go      (~200 LOC)
- linkedin_adapter.go    (~200 LOC)
- twitter_adapter.go     (~200 LOC)
```

**API Keys Required** (Zero Cost Options):
- GitHub: Free tier (5,000 req/hr)
- LinkedIn: OAuth (requires app approval, free)
- Twitter: Basic tier (1,500 tweets/month free)

---

## Phase 1: Core WASM Module (Week 1, 8-12 hours)

### Deliverables

1. **Reputation Aggregation WASM** (`modules/reputation/src/lib.rs`)
2. **Test Suite** (`modules/reputation/tests/`)
3. **Golden Vectors** (determinism validation)
4. **Build Pipeline** (Makefile integration)

### File Structure

```
modules/reputation/
├── Cargo.toml                  # Rust WASM project
├── src/
│   ├── lib.rs                  # Main entry point
│   ├── aggregation.rs          # Score aggregation logic
│   ├── weights.rs              # Weighting algorithms
│   └── types.rs                # Data structures
├── tests/
│   ├── determinism_test.rs     # Verify deterministic execution
│   └── golden_vectors.rs       # Cross-platform conformance
└── Makefile                    # Build to WASM target
```

### Implementation (lib.rs)

```rust
use serde::{Deserialize, Serialize};
use std::collections::HashMap;

#[derive(Deserialize)]
pub struct ReputationInput {
    pub user_id: String,
    pub platforms: Vec<PlatformScore>,
    pub weights: ScoreWeights,
}

#[derive(Deserialize)]
pub struct PlatformScore {
    pub platform_type: String,
    pub score: f64,
    pub weight: f64,
}

#[derive(Deserialize)]
pub struct ScoreWeights {
    pub recency: f64,
    pub volume: f64,
    pub diversity: f64,
}

#[derive(Serialize)]
pub struct ReputationOutput {
    pub trust_score: f64,
    pub confidence: f64,
    pub components: HashMap<String, ComponentScore>,
    pub timestamp: u64,
    pub deterministic_hash: String,
}

#[derive(Serialize)]
pub struct ComponentScore {
    pub score: f64,
    pub weight: f64,
}

// Deterministic aggregation function
#[no_mangle]
pub extern "C" fn aggregate_reputation(input_ptr: *const u8, input_len: usize) -> *mut u8 {
    // Parse input (deterministic JSON deserialization)
    let input_bytes = unsafe { std::slice::from_raw_parts(input_ptr, input_len) };
    let input: ReputationInput = serde_json::from_slice(input_bytes)
        .expect("Invalid input JSON");

    // Calculate weighted score (deterministic math)
    let mut weighted_sum = 0.0;
    let mut total_weight = 0.0;
    let mut components = HashMap::new();

    for platform in input.platforms {
        weighted_sum += platform.score * platform.weight;
        total_weight += platform.weight;
        components.insert(platform.platform_type.clone(), ComponentScore {
            score: platform.score,
            weight: platform.weight,
        });
    }

    let trust_score = weighted_sum / total_weight;
    let confidence = calculate_confidence(&input.platforms);

    // Generate deterministic hash
    let hash_input = format!("{}{}", input.user_id, trust_score);
    let deterministic_hash = format!("sha256:{:x}",
        sha256::digest(hash_input.as_bytes()));

    // Return output (deterministic JSON serialization)
    let output = ReputationOutput {
        trust_score,
        confidence,
        components,
        timestamp: get_deterministic_timestamp(),
        deterministic_hash,
    };

    let output_json = serde_json::to_vec(&output).expect("Serialization failed");

    // Return pointer to heap-allocated output
    Box::into_raw(output_json.into_boxed_slice()) as *mut u8
}

fn calculate_confidence(platforms: &[PlatformScore]) -> f64 {
    // Confidence increases with number of platforms
    // Max confidence = 1.0 with 5+ platforms
    (platforms.len() as f64 / 5.0).min(1.0)
}

fn get_deterministic_timestamp() -> u64 {
    // For determinism, use fixed timestamp or input-provided timestamp
    // In production, this would come from input parameters
    1696348800
}
```

### Build Configuration (Cargo.toml)

```toml
[package]
name = "trustscore-wasm"
version = "0.1.0"
edition = "2021"

[lib]
crate-type = ["cdylib"]

[dependencies]
serde = { version = "1.0", features = ["derive"] }
serde_json = "1.0"
sha256 = "1.0"

[profile.release]
opt-level = "z"     # Optimize for size
lto = true          # Link-time optimization
```

### Build Command

```bash
# Add to existing Makefile
build-trustscore-wasm:
	@echo "🔨 Building TrustScore WASM module..."
	cd modules/reputation && cargo build --target wasm32-unknown-unknown --release
	wasm-opt -Oz modules/reputation/target/wasm32-unknown-unknown/release/trustscore_wasm.wasm \
		-o artifacts/trustscore.wasm
	@echo "✅ TrustScore WASM built: $(shell du -h artifacts/trustscore.wasm)"
```

---

## Phase 2: API Integration (Week 2, 12-16 hours)

### API Handler Implementation

**Location**: `cmd/server/reputation_handlers.go` (NEW FILE)

```go
package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"ocx.local/pkg/deterministicvm"
	"ocx.local/pkg/receipt"
	"ocx.local/pkg/reputation"
)

// ReputationVerifyRequest represents incoming reputation verification request
type ReputationVerifyRequest struct {
	UserID    string                       `json:"user_id"`
	Platforms []reputation.PlatformProfile `json:"platforms"`
	Weights   *reputation.ScoreWeights     `json:"weights,omitempty"`
}

// ReputationVerifyResponse represents the verification result
type ReputationVerifyResponse struct {
	UserID       string                  `json:"user_id"`
	TrustScore   float64                 `json:"trust_score"`
	Confidence   float64                 `json:"confidence"`
	Components   map[string]interface{}  `json:"components"`
	ReceiptID    string                  `json:"receipt_id"`
	ReceiptB64   string                  `json:"receipt_b64"`
	ExpiresAt    time.Time               `json:"expires_at"`
	VerifyURL    string                  `json:"verify_url"`
}

// handleReputationVerify executes reputation verification using D-MVM
func (s *Server) handleReputationVerify(w http.ResponseWriter, r *http.Request) {
	startedAt := time.Now().UTC()

	// 1. Parse and validate request
	if r.Method != http.MethodPost {
		s.sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ReputationVerifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.UserID == "" || len(req.Platforms) == 0 {
		s.sendError(w, "user_id and platforms are required", http.StatusBadRequest)
		return
	}

	// Apply default weights if not provided
	if req.Weights == nil {
		req.Weights = reputation.DefaultWeights()
	}

	// 2. Prepare input for WASM module
	inputJSON, err := json.Marshal(map[string]interface{}{
		"user_id":   req.UserID,
		"platforms": req.Platforms,
		"weights":   req.Weights,
		"timestamp": startedAt.Unix(),
	})
	if err != nil {
		s.sendError(w, "Failed to prepare input", http.StatusInternalServerError)
		return
	}

	// 3. Load TrustScore WASM artifact
	// Hash is precomputed and stored in configuration
	trustScoreArtifactHash := reputation.GetTrustScoreWASMHash()

	// 4. Execute WASM module via D-MVM (reuse existing infrastructure!)
	result, err := deterministicvm.ExecuteArtifact(r.Context(), trustScoreArtifactHash, inputJSON)
	if err != nil {
		s.sendError(w, fmt.Sprintf("Execution failed: %v", err), http.StatusInternalServerError)
		return
	}

	// 5. Parse reputation output
	var repOutput reputation.Output
	if err := json.Unmarshal(result.Stdout, &repOutput); err != nil {
		s.sendError(w, "Failed to parse reputation output", http.StatusInternalServerError)
		return
	}

	// 6. Generate cryptographic receipt (reuse existing system!)
	receiptCore := &receipt.ReceiptCore{
		ProgramHash: trustScoreArtifactHash,
		InputHash:   sha256.Sum256(inputJSON),
		OutputHash:  sha256.Sum256(result.Stdout),
		GasUsed:     result.GasUsed,
		StartedAt:   uint64(startedAt.Unix()),
		FinishedAt:  uint64(result.EndTime.Unix()),
		IssuerID:    "trustscore-v1",
	}

	// Sign receipt with existing keystore
	coreBytes, _ := receipt.CanonicalizeCore(receiptCore)
	signature, pubKey, err := s.signer.Sign(r.Context(), s.keystore.GetActiveKey().ID, coreBytes)
	if err != nil {
		s.sendError(w, "Failed to sign receipt", http.StatusInternalServerError)
		return
	}

	receiptFull := &receipt.ReceiptFull{
		Core:       *receiptCore,
		Signature:  signature,
		HostCycles: result.HostCycles,
		HostInfo:   map[string]string{"app": "trustscore"},
	}

	fullReceiptBytes, _ := receipt.CanonicalizeFull(receiptFull)

	// 7. Store in database
	receiptID, err := s.store.SaveReceipt(r.Context(), *receiptFull, fullReceiptBytes)
	if err != nil {
		// Log but continue (storage is not critical for verification)
		fmt.Printf("Warning: Failed to store receipt: %v\n", err)
		receiptID = fmt.Sprintf("trustscore-%d", time.Now().Unix())
	}

	// 8. Store reputation verification record
	verificationID, err := s.reputationStore.SaveVerification(r.Context(), reputation.Verification{
		UserID:      req.UserID,
		TrustScore:  repOutput.TrustScore,
		Confidence:  repOutput.Confidence,
		Components:  repOutput.Components,
		ReceiptID:   receiptID,
		CreatedAt:   startedAt,
		ExpiresAt:   startedAt.Add(30 * 24 * time.Hour), // 30-day validity
	})
	if err != nil {
		fmt.Printf("Warning: Failed to store verification: %v\n", err)
	}

	// 9. Return response with badge URL
	response := ReputationVerifyResponse{
		UserID:     req.UserID,
		TrustScore: repOutput.TrustScore,
		Confidence: repOutput.Confidence,
		Components: repOutput.Components,
		ReceiptID:  receiptID,
		ReceiptB64: base64Encode(fullReceiptBytes),
		ExpiresAt:  startedAt.Add(30 * 24 * time.Hour),
		VerifyURL:  fmt.Sprintf("https://trustscore.ocx.dev/verify/%s", verificationID),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleReputationBadge generates SVG badge for display
func (s *Server) handleReputationBadge(w http.ResponseWriter, r *http.Request) {
	// Extract user ID from URL
	userID := extractPathParam(r.URL.Path, "/api/v1/reputation/badge/")

	// Fetch latest verification
	verification, err := s.reputationStore.GetLatestVerification(r.Context(), userID)
	if err != nil {
		// Return default "Unverified" badge
		w.Header().Set("Content-Type", "image/svg+xml")
		fmt.Fprintf(w, generateBadgeSVG("Unverified", "gray"))
		return
	}

	// Check expiration
	if time.Now().After(verification.ExpiresAt) {
		w.Header().Set("Content-Type", "image/svg+xml")
		fmt.Fprintf(w, generateBadgeSVG("Expired", "red"))
		return
	}

	// Generate badge based on trust score
	color := getBadgeColor(verification.TrustScore)
	label := fmt.Sprintf("Trust: %.0f/100", verification.TrustScore)

	w.Header().Set("Content-Type", "image/svg+xml")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	fmt.Fprintf(w, generateBadgeSVG(label, color))
}

func getBadgeColor(score float64) string {
	switch {
	case score >= 90: return "brightgreen"
	case score >= 75: return "green"
	case score >= 60: return "yellow"
	case score >= 40: return "orange"
	default: return "red"
	}
}

func generateBadgeSVG(label, color string) string {
	return fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="120" height="20">
		<linearGradient id="b" x2="0" y2="100%%">
			<stop offset="0" stop-color="#bbb" stop-opacity=".1"/>
			<stop offset="1" stop-opacity=".1"/>
		</linearGradient>
		<rect rx="3" width="120" height="20" fill="%s"/>
		<text x="60" y="14" fill="#fff" font-family="Verdana,sans-serif" font-size="11" text-anchor="middle">%s</text>
	</svg>`, color, label)
}
```

---

## Phase 3: Platform Adapters (Week 3, 16-20 hours)

### GitHub Adapter

**Location**: `pkg/reputation/adapters/github.go`

```go
package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type GitHubAdapter struct {
	apiKey     string
	httpClient *http.Client
}

func NewGitHubAdapter(apiKey string) *GitHubAdapter {
	return &GitHubAdapter{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (a *GitHubAdapter) FetchProfile(ctx context.Context, username string) (*Profile, error) {
	// Fetch user profile
	userURL := fmt.Sprintf("https://api.github.com/users/%s", username)
	user, err := a.fetchJSON(ctx, userURL)
	if err != nil {
		return nil, err
	}

	// Fetch user events (for activity score)
	eventsURL := fmt.Sprintf("https://api.github.com/users/%s/events/public", username)
	events, err := a.fetchJSONArray(ctx, eventsURL)
	if err != nil {
		return nil, err
	}

	// Fetch repositories (for contribution score)
	reposURL := fmt.Sprintf("https://api.github.com/users/%s/repos?sort=updated", username)
	repos, err := a.fetchJSONArray(ctx, reposURL)
	if err != nil {
		return nil, err
	}

	return &Profile{
		Platform:   "github",
		Username:   username,
		Followers:  getInt(user, "followers"),
		Following:  getInt(user, "following"),
		PublicRepos: getInt(user, "public_repos"),
		CreatedAt:  parseGitHubDate(getString(user, "created_at")),
		UpdatedAt:  time.Now(),
		Metrics: map[string]interface{}{
			"total_stars":   calculateTotalStars(repos),
			"total_forks":   calculateTotalForks(repos),
			"recent_activity": len(events),
			"contribution_streak": calculateStreak(events),
		},
	}, nil
}

func (a *GitHubAdapter) CalculateScore(profile *Profile) (float64, error) {
	// Scoring algorithm
	var score float64

	// Account age (max 20 points)
	accountAge := time.Since(profile.CreatedAt).Hours() / 24 / 365
	score += min(accountAge*4, 20)

	// Follower ratio (max 15 points)
	if profile.Following > 0 {
		ratio := float64(profile.Followers) / float64(profile.Following)
		score += min(ratio*3, 15)
	}

	// Public repos (max 15 points)
	score += min(float64(profile.PublicRepos)/2, 15)

	// Stars received (max 20 points)
	stars := getFloat(profile.Metrics, "total_stars")
	score += min(stars/50, 20)

	// Recent activity (max 15 points)
	activity := getFloat(profile.Metrics, "recent_activity")
	score += min(activity/2, 15)

	// Contribution streak (max 15 points)
	streak := getFloat(profile.Metrics, "contribution_streak")
	score += min(streak/4, 15)

	return min(score, 100), nil
}

func (a *GitHubAdapter) fetchJSON(ctx context.Context, url string) (map[string]interface{}, error) {
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+a.apiKey)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("GitHub API error: %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result, nil
}

// Helper functions...
func calculateTotalStars(repos []interface{}) float64 {
	total := 0.0
	for _, repo := range repos {
		if r, ok := repo.(map[string]interface{}); ok {
			total += getFloat(r, "stargazers_count")
		}
	}
	return total
}

func calculateStreak(events []interface{}) float64 {
	// Calculate contribution streak from events
	// Implementation details...
	return float64(len(events))
}
```

### LinkedIn & Twitter Adapters
Similar structure to GitHub adapter, implementing platform-specific logic.

---

## Phase 4: Database & Storage (Week 4, 8-12 hours)

### Repository Implementation

**Location**: `pkg/reputation/repository.go`

```go
package reputation

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) SaveVerification(ctx context.Context, v Verification) (string, error) {
	componentsJSON, _ := json.Marshal(v.Components)

	query := `
		INSERT INTO reputation_verifications
		(user_id, trust_score, confidence, components, receipt_id, created_at, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`

	var id string
	err := r.db.QueryRow(ctx, query,
		v.UserID, v.TrustScore, v.Confidence, componentsJSON,
		v.ReceiptID, v.CreatedAt, v.ExpiresAt,
	).Scan(&id)

	return id, err
}

func (r *Repository) GetLatestVerification(ctx context.Context, userID string) (*Verification, error) {
	query := `
		SELECT id, user_id, trust_score, confidence, components, receipt_id, created_at, expires_at
		FROM reputation_verifications
		WHERE user_id = $1 AND expires_at > NOW()
		ORDER BY created_at DESC
		LIMIT 1
	`

	var v Verification
	var componentsJSON []byte

	err := r.db.QueryRow(ctx, query, userID).Scan(
		&v.ID, &v.UserID, &v.TrustScore, &v.Confidence, &componentsJSON,
		&v.ReceiptID, &v.CreatedAt, &v.ExpiresAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	json.Unmarshal(componentsJSON, &v.Components)
	return &v, nil
}

func (r *Repository) GetVerificationHistory(ctx context.Context, userID string, limit int) ([]Verification, error) {
	query := `
		SELECT id, user_id, trust_score, confidence, components, receipt_id, created_at, expires_at
		FROM reputation_verifications
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`

	rows, err := r.db.Query(ctx, query, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []Verification
	for rows.Next() {
		var v Verification
		var componentsJSON []byte

		err := rows.Scan(&v.ID, &v.UserID, &v.TrustScore, &v.Confidence,
			&componentsJSON, &v.ReceiptID, &v.CreatedAt, &v.ExpiresAt)
		if err != nil {
			continue
		}

		json.Unmarshal(componentsJSON, &v.Components)
		results = append(results, v)
	}

	return results, nil
}
```

---

## Zero-Capital PLG Strategy

### Distribution Channels (100% Organic)

#### 1. GitHub Profile Badges
**Implementation**: Generate embeddable SVG badges
**Placement**: README.md files
**Virality**: Developers copy badges from popular repos

```markdown
[![TrustScore](https://trustscore.ocx.dev/badge/alice)](https://trustscore.ocx.dev/verify/alice)
```

#### 2. LinkedIn Profile Integration
**Implementation**: Verifiable credentials API
**Placement**: LinkedIn "Licenses & Certifications" section
**Virality**: Professional network sees verified credentials

#### 3. Twitter/X Profile Link
**Implementation**: Verification URL in bio
**Placement**: Profile description
**Virality**: Every follower sees verification

#### 4. Developer Forums
**Implementation**: Forum signature integration
**Placement**: Stack Overflow, Reddit, HackerNews profiles
**Virality**: High-visibility in technical communities

### Pricing Tiers

#### Free Tier
- 1 verification per month
- 3 platform connections max
- Public badge only
- 30-day validity

#### Pro Tier ($9/mo)
- Unlimited verifications
- 10 platform connections
- Custom badge styles
- 90-day validity
- API access (100 req/day)

#### Enterprise Tier ($29/mo)
- Everything in Pro
- Unlimited platform connections
- White-label badges
- 1-year validity
- API access (1000 req/day)
- Priority support

### Revenue Projections (Conservative)

**Month 1-3**: Free tier only, focus on growth
- Target: 100 users (organic GitHub/LinkedIn)

**Month 4-6**: Introduce Pro tier
- Conversion: 5% (5 paid users)
- MRR: $45

**Month 7-12**: Scale to 1,000 users
- Conversion: 5% (50 paid users)
- MRR: $450

**Year 2**: Scale to 10,000 users
- Conversion: 5% (500 paid users)
- MRR: $4,500
- ARR: $54,000

---

## Risk Mitigation

### Technical Risks

1. **WASM Determinism**: Mitigated by extensive testing (golden vectors)
2. **Platform API Changes**: Versioned adapters, graceful degradation
3. **Rate Limiting**: Caching, intelligent request batching
4. **Database Scale**: Existing OCX infrastructure handles millions of receipts

### Business Risks

1. **Low Adoption**: Mitigated by free tier, viral badge mechanism
2. **Platform API Costs**: Free tiers sufficient for MVP (5K GitHub, 1.5K Twitter)
3. **Competition**: First-mover with cryptographic proofs (unique value prop)
4. **Fraud**: Ed25519 signatures prevent score manipulation

---

## Success Metrics

### Technical KPIs
- **Determinism**: 100% identical scores for identical inputs
- **Performance**: <100ms end-to-end verification
- **Uptime**: 99.9% (leverage existing OCX infrastructure)
- **Verification Rate**: 95%+ successful verifications

### Business KPIs
- **User Growth**: 10% MoM organic growth
- **Conversion Rate**: 5% free → paid
- **Badge Embedding**: 30% of users embed badges
- **API Usage**: 50% of Pro users use API
- **Churn**: <5% monthly churn

---

## Timeline & Milestones

### Week 1-2: Foundation
- [x] Complete audit of OCX infrastructure
- [ ] Build WASM reputation module
- [ ] Add database schema
- [ ] Create API endpoints

### Week 3-4: Platform Integration
- [ ] GitHub adapter implementation
- [ ] LinkedIn adapter implementation
- [ ] Twitter adapter implementation
- [ ] Badge generation system

### Week 5-6: Testing & Polish
- [ ] End-to-end testing
- [ ] Load testing (reuse OCX load tests)
- [ ] Security audit (reuse OCX security framework)
- [ ] Documentation

### Week 7-8: Soft Launch
- [ ] Deploy to production
- [ ] Onboard 10 beta users
- [ ] Gather feedback
- [ ] Iterate on UX

### Week 9-12: Growth Phase
- [ ] Public launch on HackerNews/Reddit
- [ ] Integration with OCX fraud proof marketing
- [ ] Track metrics, optimize conversion
- [ ] Plan Pro tier launch

---

## Conclusion

This integration leverages **100% of existing OCX infrastructure** while adding a complementary consumer product. The technical foundation (D-MVM, receipts, API) is production-ready, minimizing development risk. The zero-capital PLG strategy ensures sustainable growth without upfront investment.

**Total New Code**: ~2,000 LOC (vs. 50,000+ existing OCX LOC)
**Code Reuse**: 96%+
**Time to MVP**: 6 weeks
**Time to Revenue**: 12 weeks

The dual-market approach (L2 fraud proofs + consumer reputation) de-risks enterprise sales while demonstrating protocol versatility and building brand awareness in developer communities.
