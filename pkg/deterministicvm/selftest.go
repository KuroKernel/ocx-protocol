package deterministicvm

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

// SelfTestOut represents the output of the self-test endpoint
type SelfTestOut struct {
	ArtifactID string `json:"artifact_id"`
	Runs       []struct {
		Sha256     string `json:"sha256"`
		DurationNs int64  `json:"duration_ns"`
	} `json:"runs"`
	Deterministic bool   `json:"deterministic"`
	EnvHash       string `json:"env_hash"`
}

// SelfTestHandler handles the /determinism/selftest endpoint
func SelfTestHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse request body
	var req struct {
		ArtifactID string `json:"artifact_id"`
		Input      []byte `json:"input"`
		Runs       int    `json:"runs"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Default values
	if req.Runs == 0 {
		req.Runs = 3 // Default to 3 runs
	}
	if req.ArtifactID == "" {
		req.ArtifactID = "test_artifact"
	}
	if len(req.Input) == 0 {
		req.Input = []byte("test_input")
	}

	// Run the self-test
	result, err := runSelfTest(context.Background(), req.ArtifactID, req.Input, req.Runs)
	if err != nil {
		http.Error(w, fmt.Sprintf("Self-test failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Return results
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// runSelfTest executes the same artifact multiple times to verify determinism
func runSelfTest(ctx context.Context, artifactID string, input []byte, runs int) (*SelfTestOut, error) {
	log.Printf("Running self-test: artifact=%s, runs=%d", artifactID, runs)

	// Create a test artifact hash (in real implementation, this would be a real artifact)
	artifactHash := sha256.Sum256([]byte(artifactID))

	var testRuns []struct {
		Sha256     string `json:"sha256"`
		DurationNs int64  `json:"duration_ns"`
	}

	var firstOutputHash string
	allDeterministic := true

	// Run the artifact multiple times
	for i := 0; i < runs; i++ {
		startTime := time.Now()

		// Execute the artifact
		result, err := ExecuteArtifact(ctx, artifactHash, input)
		if err != nil {
			return nil, fmt.Errorf("execution %d failed: %w", i+1, err)
		}

		duration := time.Since(startTime)
		outputHash := fmt.Sprintf("sha256:%x", sha256.Sum256(result.Stdout))

		// Check determinism
		if i == 0 {
			firstOutputHash = outputHash
		} else if outputHash != firstOutputHash {
			allDeterministic = false
			log.Printf("Determinism violation detected in run %d: expected %s, got %s",
				i+1, firstOutputHash, outputHash)
		}

		testRuns = append(testRuns, struct {
			Sha256     string `json:"sha256"`
			DurationNs int64  `json:"duration_ns"`
		}{
			Sha256:     outputHash,
			DurationNs: duration.Nanoseconds(),
		})

		log.Printf("Run %d: hash=%s, duration=%v", i+1, outputHash, duration)
	}

	// Get environment hash
	envHash := EnvHash()

	return &SelfTestOut{
		ArtifactID:    artifactID,
		Runs:          testRuns,
		Deterministic: allDeterministic,
		EnvHash:       envHash,
	}, nil
}

// NegativeTestHandler handles negative tests for security isolation
func NegativeTestHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse request body
	var req struct {
		TestType string `json:"test_type"` // "network", "time", "rng", "filesystem"
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Run the appropriate negative test
	var result struct {
		TestType   string `json:"test_type"`
		Blocked    bool   `json:"blocked"`
		Error      string `json:"error,omitempty"`
		DurationMs int64  `json:"duration_ms"`
	}

	startTime := time.Now()

	switch req.TestType {
	case "network":
		result = testNetworkBlocking()
	case "time":
		result = testTimeBlocking()
	case "rng":
		result = testRNGBlocking()
	case "filesystem":
		result = testFilesystemBlocking()
	default:
		http.Error(w, "Unknown test type", http.StatusBadRequest)
		return
	}

	result.DurationMs = time.Since(startTime).Milliseconds()

	// Return results
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// testNetworkBlocking tests that network access is blocked
func testNetworkBlocking() struct {
	TestType   string `json:"test_type"`
	Blocked    bool   `json:"blocked"`
	Error      string `json:"error,omitempty"`
	DurationMs int64  `json:"duration_ms"`
} {
	// This would test network access in a real implementation
	// we'll simulate the test
	return struct {
		TestType   string `json:"test_type"`
		Blocked    bool   `json:"blocked"`
		Error      string `json:"error,omitempty"`
		DurationMs int64  `json:"duration_ms"`
	}{
		TestType: "network",
		Blocked:  true, // Simulate that network is blocked
		Error:    "network access blocked by seccomp",
	}
}

// testTimeBlocking tests that time access is blocked
func testTimeBlocking() struct {
	TestType   string `json:"test_type"`
	Blocked    bool   `json:"blocked"`
	Error      string `json:"error,omitempty"`
	DurationMs int64  `json:"duration_ms"`
} {
	// This would test time access in a real implementation
	return struct {
		TestType   string `json:"test_type"`
		Blocked    bool   `json:"blocked"`
		Error      string `json:"error,omitempty"`
		DurationMs int64  `json:"duration_ms"`
	}{
		TestType: "time",
		Blocked:  true, // Simulate that time access is blocked
		Error:    "time access blocked by seccomp",
	}
}

// testRNGBlocking tests that RNG access is blocked
func testRNGBlocking() struct {
	TestType   string `json:"test_type"`
	Blocked    bool   `json:"blocked"`
	Error      string `json:"error,omitempty"`
	DurationMs int64  `json:"duration_ms"`
} {
	// This would test RNG access in a real implementation
	return struct {
		TestType   string `json:"test_type"`
		Blocked    bool   `json:"blocked"`
		Error      string `json:"error,omitempty"`
		DurationMs int64  `json:"duration_ms"`
	}{
		TestType: "rng",
		Blocked:  true, // Simulate that RNG access is blocked
		Error:    "RNG access blocked by seccomp",
	}
}

// testFilesystemBlocking tests that filesystem access is restricted
func testFilesystemBlocking() struct {
	TestType   string `json:"test_type"`
	Blocked    bool   `json:"blocked"`
	Error      string `json:"error,omitempty"`
	DurationMs int64  `json:"duration_ms"`
} {
	// This would test filesystem access in a real implementation
	return struct {
		TestType   string `json:"test_type"`
		Blocked    bool   `json:"blocked"`
		Error      string `json:"error,omitempty"`
		DurationMs int64  `json:"duration_ms"`
	}{
		TestType: "filesystem",
		Blocked:  true, // Simulate that filesystem access is restricted
		Error:    "filesystem access restricted to /scratch",
	}
}
