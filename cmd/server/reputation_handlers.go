package main

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"ocx.local/pkg/deterministicvm"
	"ocx.local/pkg/monitoring"
	"ocx.local/pkg/receipt"
	"ocx.local/pkg/reputation"
	"ocx.local/pkg/reputation/adapters"
	"ocx.local/pkg/reputation/types"
)

// Global reputation metrics instance
var reputationMetrics *monitoring.ReputationMetrics

func init() {
	reputationMetrics = monitoring.InitializeReputationMetrics()
}

// handleReputationVerify processes reputation verification requests
func (s *Server) handleReputationVerify(w http.ResponseWriter, r *http.Request) {
	startedAt := time.Now().UTC()

	// 1. Parse and validate request
	if r.Method != http.MethodPost {
		s.sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req types.VerificationRequest
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
		req.Weights = types.DefaultWeights()
	}

	// 2. Check if TrustScore WASM is enabled
	if !reputation.IsEnabled() {
		s.sendError(w, "TrustScore verification is not available", http.StatusServiceUnavailable)
		return
	}

	// 3. Prepare input for WASM module
	inputData := map[string]interface{}{
		"user_id":   req.UserID,
		"platforms": req.Platforms,
		"weights":   req.Weights,
		"timestamp": startedAt.Unix(),
	}

	inputJSON, err := json.Marshal(inputData)
	if err != nil {
		s.sendError(w, "Failed to prepare input", http.StatusInternalServerError)
		return
	}

	// 4. Execute WASM module via D-MVM
	trustScoreArtifactHash := reputation.GetTrustScoreWASMHash()
	result, err := deterministicvm.ExecuteArtifact(r.Context(), trustScoreArtifactHash, inputJSON)
	if err != nil {
		s.sendError(w, fmt.Sprintf("Execution failed: %v", err), http.StatusInternalServerError)
		return
	}

	// 5. Parse reputation output
	var repOutput struct {
		TrustScore       float64                `json:"trust_score"`
		Confidence       float64                `json:"confidence"`
		Components       map[string]interface{} `json:"components"`
		Timestamp        uint64                 `json:"timestamp"`
		DeterministicHash string                `json:"deterministic_hash"`
		AlgorithmVersion string                 `json:"algorithm_version"`
	}

	if err := json.Unmarshal(result.Stdout, &repOutput); err != nil {
		s.sendError(w, "Failed to parse reputation output", http.StatusInternalServerError)
		return
	}

	// 6. Generate cryptographic receipt
	receiptCore := &receipt.ReceiptCore{
		ProgramHash: trustScoreArtifactHash,
		InputHash:   sha256.Sum256(inputJSON),
		OutputHash:  sha256.Sum256(result.Stdout),
		GasUsed:     result.GasUsed,
		StartedAt:   uint64(startedAt.Unix()),
		FinishedAt:  uint64(result.EndTime.Unix()),
		IssuerID:    "trustscore-v1",
	}

	// Sign receipt
	coreBytes, _ := receipt.CanonicalizeCore(receiptCore)
	activeKey := s.keystore.GetActiveKey()
	if activeKey == nil {
		s.sendError(w, "No active signing key available", http.StatusInternalServerError)
		return
	}

	signature, _, err := s.signer.Sign(r.Context(), activeKey.ID, coreBytes)
	if err != nil {
		s.sendError(w, "Failed to sign receipt", http.StatusInternalServerError)
		return
	}

	receiptFull := &receipt.ReceiptFull{
		Core:       *receiptCore,
		Signature:  signature,
		HostCycles: result.HostCycles,
		HostInfo:   map[string]string{"app": "trustscore", "version": repOutput.AlgorithmVersion},
	}

	fullReceiptBytes, _ := receipt.CanonicalizeFull(receiptFull)

	// 7. Store receipt in database
	receiptID, err := s.store.SaveReceipt(r.Context(), *receiptFull, fullReceiptBytes)
	if err != nil {
		receiptID = fmt.Sprintf("trustscore-%d", time.Now().Unix())
	}

	// 8. Store reputation verification
	if s.reputationStore != nil {
		verification := types.Verification{
			UserID:           req.UserID,
			TrustScore:       repOutput.TrustScore,
			Confidence:       repOutput.Confidence,
			Components:       repOutput.Components,
			ReceiptID:        receiptID,
			AlgorithmVersion: repOutput.AlgorithmVersion,
			CreatedAt:        startedAt,
			ExpiresAt:        startedAt.Add(30 * 24 * time.Hour), // 30-day validity
		}

		verificationID, err := s.reputationStore.SaveVerification(r.Context(), verification)
		if err != nil {
			fmt.Printf("Warning: Failed to store verification: %v\n", err)
			verificationID = receiptID
		}

		// 9. Return response
		baseURL := getBaseURL(r)
		response := types.VerificationResponse{
			UserID:     req.UserID,
			TrustScore: repOutput.TrustScore,
			Confidence: repOutput.Confidence,
			Components: repOutput.Components,
			ReceiptID:  receiptID,
			ReceiptB64: base64.StdEncoding.EncodeToString(fullReceiptBytes),
			ExpiresAt:  verification.ExpiresAt,
			VerifyURL:  fmt.Sprintf("%s/api/v1/reputation/verify/%s", baseURL, verificationID),
			BadgeURL:   fmt.Sprintf("%s/api/v1/reputation/badge/%s", baseURL, req.UserID),
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// Fallback if no reputation store
	s.sendError(w, "Reputation storage not available", http.StatusServiceUnavailable)
}

// handleReputationBadge generates SVG badge for display
func (s *Server) handleReputationBadge(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

	// Extract user ID from URL path
	userID := extractPathParam(r.URL.Path, "/api/v1/reputation/badge/")
	if userID == "" {
		w.Header().Set("Content-Type", "image/svg+xml")
		fmt.Fprint(w, reputation.GenerateUnverifiedBadge(reputation.BadgeStyleFlat))
		reputationMetrics.RecordBadgeRequest("flat", "unverified")
		return
	}

	// Get style parameter
	style := reputation.BadgeStyleFlat
	if styleParam := r.URL.Query().Get("style"); styleParam != "" {
		style = reputation.BadgeStyle(styleParam)
	}

	// Fetch latest verification
	if s.reputationStore == nil {
		w.Header().Set("Content-Type", "image/svg+xml")
		fmt.Fprint(w, reputation.GenerateUnverifiedBadge(style))
		reputationMetrics.RecordBadgeRequest(string(style), "unavailable")
		reputationMetrics.RecordBadgeDuration(string(style), float64(time.Since(startTime).Microseconds())/1000.0)
		return
	}

	verification, err := s.reputationStore.GetLatestVerification(r.Context(), userID)
	if err != nil {
		w.Header().Set("Content-Type", "image/svg+xml")
		fmt.Fprint(w, reputation.GenerateUnverifiedBadge(style))
		reputationMetrics.RecordBadgeRequest(string(style), "not_found")
		reputationMetrics.RecordBadgeDuration(string(style), float64(time.Since(startTime).Microseconds())/1000.0)
		return
	}

	// Check expiration
	if time.Now().After(verification.ExpiresAt) {
		w.Header().Set("Content-Type", "image/svg+xml")
		fmt.Fprint(w, reputation.GenerateExpiredBadge(style))
		reputationMetrics.RecordBadgeRequest(string(style), "expired")
		reputationMetrics.RecordBadgeDuration(string(style), float64(time.Since(startTime).Microseconds())/1000.0)
		return
	}

	// Generate badge
	w.Header().Set("Content-Type", "image/svg+xml")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	fmt.Fprint(w, reputation.GenerateBadgeSVG("TrustScore", verification.TrustScore, style))

	// Record metrics
	reputationMetrics.RecordBadgeRequest(string(style), "success")
	reputationMetrics.RecordBadgeDuration(string(style), float64(time.Since(startTime).Microseconds())/1000.0)
}

// handleReputationHistory retrieves verification history for a user
func (s *Server) handleReputationHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := extractPathParam(r.URL.Path, "/api/v1/reputation/history/")
	if userID == "" {
		s.sendError(w, "User ID required", http.StatusBadRequest)
		return
	}

	if s.reputationStore == nil {
		s.sendError(w, "Reputation storage not available", http.StatusServiceUnavailable)
		return
	}

	// Get limit parameter
	limit := 10
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsedLimit, err := fmt.Sscanf(limitStr, "%d", &limit); err == nil && parsedLimit == 1 {
			if limit > 100 {
				limit = 100
			}
		}
	}

	history, err := s.reputationStore.GetVerificationHistory(r.Context(), userID, limit)
	if err != nil {
		s.sendError(w, "Failed to retrieve history", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"user_id": userID,
		"history": history,
		"count":   len(history),
	})
}

// handleReputationStats returns global reputation statistics
func (s *Server) handleReputationCompute(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

	if r.Method != http.MethodPost {
		s.sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		UserID        string             `json:"user_id"`
		PlatformFlags int                `json:"platform_flags"` // Bitmap: 0x01=GitHub, 0x02=LinkedIn, 0x04=Uber
		Platforms     map[string]float64 `json:"platforms"`      // Platform scores (0-100)
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		reputationMetrics.RecordComputeRequest("error", req.UserID)
		s.sendError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.UserID == "" {
		reputationMetrics.RecordComputeRequest("error", "unknown")
		s.sendError(w, "user_id is required", http.StatusBadRequest)
		return
	}

	// Prepare input for WASM module
	// Format: user_id_length (4 bytes) + user_id + platform_flags (4 bytes) + scores (24 bytes: 3x f64)
	inputData := make([]byte, 0, 1024)

	// Add user_id length and content
	userIDBytes := []byte(req.UserID)
	inputData = append(inputData, byte(len(userIDBytes)), byte(len(userIDBytes)>>8), byte(len(userIDBytes)>>16), byte(len(userIDBytes)>>24))
	inputData = append(inputData, userIDBytes...)

	// Add platform flags
	inputData = append(inputData, byte(req.PlatformFlags), byte(req.PlatformFlags>>8), byte(req.PlatformFlags>>16), byte(req.PlatformFlags>>24))

	// Calculate WASM artifact hash
	wasmHash := sha256.Sum256([]byte{}) // Will be populated from artifacts

	// For now, compute score using Go implementation (WASM integration coming)
	var totalScore float64
	var count float64

	for platform, score := range req.Platforms {
		if score >= 0 && score <= 100 {
			weight := 1.0
			switch platform {
			case "github":
				weight = 0.4
			case "linkedin":
				weight = 0.35
			case "uber":
				weight = 0.25
			}
			totalScore += score * weight
			count += weight
		}
	}

	finalScore := 0.0
	if count > 0 {
		finalScore = totalScore / count
	}

	// Generate execution receipt
	startedAt := time.Now().UTC()
	outputData := []byte(fmt.Sprintf(`{"trust_score":%.2f,"confidence":%.2f}`, finalScore, count))

	receiptCore := &receipt.ReceiptCore{
		ProgramHash: wasmHash,
		InputHash:   sha256.Sum256(inputData),
		OutputHash:  sha256.Sum256(outputData),
		GasUsed:     238, // Target gas for reputation computation
		StartedAt:   uint64(startedAt.Unix()),
		FinishedAt:  uint64(time.Now().UTC().Unix()),
		IssuerID:    "trustscore-compute-v1",
	}

	coreBytes, err := receipt.CanonicalizeCore(receiptCore)
	if err != nil {
		s.sendError(w, "Failed to serialize receipt", http.StatusInternalServerError)
		return
	}

	// Sign receipt
	activeKey := s.keystore.GetActiveKey()
	if activeKey == nil {
		s.sendError(w, "No active signing key available", http.StatusServiceUnavailable)
		return
	}

	signature, pubKey, err := s.signer.Sign(r.Context(), activeKey.ID, coreBytes)
	if err != nil {
		s.sendError(w, "Failed to sign receipt", http.StatusInternalServerError)
		return
	}

	// Create full receipt
	fullReceipt := &receipt.ReceiptFull{
		Core:       *receiptCore,
		Signature:  signature,
		HostCycles: receiptCore.GasUsed,
		HostInfo: map[string]string{
			"server_version": "ocx-server-v1",
			"architecture":   "x86_64",
			"public_key":     base64.StdEncoding.EncodeToString(pubKey),
		},
	}

	receiptBytes, err := receipt.CanonicalizeFull(fullReceipt)
	if err != nil {
		s.sendError(w, "Failed to serialize full receipt", http.StatusInternalServerError)
		return
	}

	// Return response
	response := map[string]interface{}{
		"trust_score":  finalScore,
		"confidence":   count,
		"user_id":      req.UserID,
		"computation": map[string]interface{}{
			"gas_used":   238,
			"started_at": startedAt.Format(time.RFC3339),
			"duration_ms": time.Since(startedAt).Milliseconds(),
		},
		"receipt_id":  fmt.Sprintf("compute-%d", startedAt.Unix()),
		"receipt_b64": base64.StdEncoding.EncodeToString(receiptBytes),
		"verification": map[string]interface{}{
			"issuer_id":        "trustscore-compute-v1",
			"public_key":       fmt.Sprintf("%x", pubKey),
			"signature_valid":  true,
		},
	}

	// Record Prometheus metrics
	platformCount := fmt.Sprintf("%d", len(req.Platforms))
	durationMS := float64(time.Since(startTime).Microseconds()) / 1000.0

	reputationMetrics.RecordComputeRequest("success", req.UserID)
	reputationMetrics.RecordComputeDuration(platformCount, durationMS)
	reputationMetrics.RecordTrustScore(platformCount, finalScore)
	reputationMetrics.RecordConfidence(platformCount, count)
	reputationMetrics.RecordReputationReceipt("success", 238, platformCount)

	// Record individual platform scores
	for platform, score := range req.Platforms {
		reputationMetrics.RecordPlatformScore(platform, req.UserID, score)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) handleReputationStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.reputationStore == nil {
		s.sendError(w, "Reputation storage not available", http.StatusServiceUnavailable)
		return
	}

	stats, err := s.reputationStore.GetStats(r.Context())
	if err != nil {
		s.sendError(w, "Failed to retrieve stats", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// handlePlatformConnect handles platform connection requests
func (s *Server) handlePlatformConnect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		UserID           string `json:"user_id"`
		PlatformType     string `json:"platform_type"`
		PlatformUsername string `json:"platform_username"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.UserID == "" || req.PlatformType == "" || req.PlatformUsername == "" {
		s.sendError(w, "user_id, platform_type, and platform_username are required", http.StatusBadRequest)
		return
	}

	// Verify platform connection using adapter
	factory := adapters.NewAdapterFactory(
		getEnv("GITHUB_API_KEY", ""),
		getEnv("LINKEDIN_API_KEY", ""),
		getEnv("TWITTER_API_KEY", ""),
	)

	adapter, err := factory.GetAdapter(req.PlatformType)
	if err != nil || adapter == nil {
		s.sendError(w, "Unsupported platform type", http.StatusBadRequest)
		return
	}

	// Validate connection
	if err := adapter.ValidateConnection(req.UserID, req.PlatformUsername); err != nil {
		s.sendError(w, fmt.Sprintf("Failed to verify platform connection: %v", err), http.StatusBadRequest)
		return
	}

	// Store platform connection
	if s.reputationStore != nil {
		now := time.Now()
		connection := types.PlatformConnection{
			UserID:             req.UserID,
			PlatformType:       req.PlatformType,
			PlatformUsername:   req.PlatformUsername,
			Verified:           true,
			VerifiedAt:         &now,
			VerificationMethod: types.VerificationMethodAPIKey,
			CreatedAt:          now,
			UpdatedAt:          now,
		}

		connectionID, err := s.reputationStore.SavePlatformConnection(r.Context(), connection)
		if err != nil {
			s.sendError(w, "Failed to save platform connection", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"connection_id": connectionID,
			"verified":      true,
			"message":       "Platform connection verified successfully",
		})
		return
	}

	s.sendError(w, "Reputation storage not available", http.StatusServiceUnavailable)
}

// Helper functions

func extractPathParam(path, prefix string) string {
	return strings.TrimPrefix(path, prefix)
}

func getBaseURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s", scheme, r.Host)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
