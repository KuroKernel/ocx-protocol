// internal/engine/unified_updated.go
package engine

import (
	"database/sql"
	"fmt"
	"log"
	"time"
	
	// Import our components
	"ocx.local/internal/query/ocxql"
	"ocx.local/internal/consensus/telemetry"
	"ocx.local/internal/tee"
	"ocx.local/internal/tokenomics"
	"ocx.local/internal/zkproofs"
)

// OCXProtocolEngine is the unified engine that integrates all OCX Protocol components
type OCXProtocolEngine struct {
	// Core components
	OCXQL        *ocxql.OCXQLEngine
	TelemetryConsensus *telemetry.TelemetryConsensusEngine
	TEEAttestation *tee.TEEMeasurementEngine
	USDTokenomics *tokenomics.USDTokenomicsEngine
	ZKProofs     *zkproofs.ZKProofEngine
	
	// Database connection
	DB *sql.DB
	
	// Configuration
	Config *EngineConfig
}

// EngineConfig contains configuration for the unified engine
type EngineConfig struct {
	// OCX-QL settings
	OCXQLEnabled bool
	
	// Telemetry consensus settings
	TelemetryConsensusEnabled bool
	ByzantineTolerance float64
	
	// TEE attestation settings
	TEEAttestationEnabled bool
	TEEType string
	
	// USD tokenomics settings
	USDTokenomicsEnabled bool
	TransactionFeeBPS int
	VerifierFeeShare float64
	
	// ZK proofs settings
	ZKProofsEnabled bool
	PrivacyLevel string
}

// NewOCXProtocolEngine creates a new unified OCX Protocol engine
func NewOCXProtocolEngine(db *sql.DB, config *EngineConfig) *OCXProtocolEngine {
	engine := &OCXProtocolEngine{
		DB:     db,
		Config: config,
	}
	
	// Initialize components based on configuration
	if config.OCXQLEnabled {
		engine.OCXQL = ocxql.NewOCXQLEngine(db)
		log.Println("OCX-QL DSL engine initialized")
	}
	
	if config.TelemetryConsensusEnabled {
		engine.TelemetryConsensus = telemetry.NewTelemetryConsensusEngine()
		log.Println("Telemetry consensus engine initialized")
	}
	
	if config.TEEAttestationEnabled {
		teeType := tee.IntelSGX // Default
		switch config.TEEType {
		case "amd_sev":
			teeType = tee.AMDSEV
		case "aws_nitro":
			teeType = tee.AWSNitro
		case "arm_trustzone":
			teeType = tee.ARMTrustZone
		}
		engine.TEEAttestation = tee.NewTEEMeasurementEngine(teeType)
		log.Printf("TEE attestation engine initialized with %s", config.TEEType)
	}
	
	if config.USDTokenomicsEnabled {
		engine.USDTokenomics = tokenomics.NewUSDTokenomicsEngine()
		log.Println("USD tokenomics engine initialized")
	}
	
	if config.ZKProofsEnabled {
		engine.ZKProofs = zkproofs.NewZKProofEngine()
		if config.PrivacyLevel != "" {
			engine.ZKProofs.SetPrivacyLevel(config.PrivacyLevel)
		}
		log.Println("ZK proofs engine initialized")
	}
	
	log.Println("OCX Protocol unified engine initialized successfully")
	return engine
}

// ProcessComputeRequest processes a complete compute request through all systems
func (e *OCXProtocolEngine) ProcessComputeRequest(request *ComputeRequest) (*ComputeResponse, error) {
	startTime := time.Now()
	
	log.Printf("Processing compute request: %s", request.WorkloadID)
	
	// Step 1: Parse OCX-QL query if provided
	var parsedQuery *ocxql.OCXQLQuery
	if request.OCXQLQuery != "" && e.OCXQL != nil {
		var err error
		result, err := e.OCXQL.Execute(request.OCXQLQuery)
		if err != nil {
			return nil, fmt.Errorf("OCX-QL execution failed: %w", err)
		}
		parsedQuery = result.Query
		if err != nil {
			return nil, fmt.Errorf("OCX-QL parsing failed: %w", err)
		}
		log.Printf("OCX-QL query parsed successfully")
	}
	
	// Step 2: Start workload with telemetry consensus
	if e.TelemetryConsensus != nil {
		workload := &telemetry.ComputeWorkload{
			WorkloadID:      request.WorkloadID,
			ProviderID:      request.ProviderID,
			CustomerID:      request.CustomerID,
			ResourceSpec:    request.ResourceSpec,
			ExpectedDuration: request.ExpectedDuration,
			SLARequirements: request.SLARequirements,
			StartTime:       time.Now(),
		}
		
		if err := e.TelemetryConsensus.StartWorkload(workload); err != nil {
			return nil, fmt.Errorf("failed to start workload: %w", err)
		}
		log.Printf("Workload started with telemetry consensus")
	}
	
	// Step 3: Generate TEE attestation if enabled
	var teeMeasurement *tee.ComputeMeasurement
	if e.TEEAttestation != nil && request.RequireTEEAttestation {
		measurement, err := e.TEEAttestation.CreateMeasurement(request.WorkloadID, request.ResourceSpec)
		if err != nil {
			return nil, fmt.Errorf("TEE attestation failed: %w", err)
		}
		teeMeasurement = measurement
		log.Printf("TEE attestation generated")
	}
	
	// Step 4: Process USD payment
	var paymentResult *tokenomics.USDPaymentResult
	if e.USDTokenomics != nil {
		payment, err := e.USDTokenomics.ProcessUSDPayment(request.CustomerID, request.ProviderID, request.USDCost)
		if err != nil {
			return nil, fmt.Errorf("USD payment processing failed: %w", err)
		}
		paymentResult = payment
		log.Printf("USD payment processed: $%.2f", request.USDCost)
	}
	
	// Step 5: Generate ZK proof if required
	var zkProof *zkproofs.ZKProof
	if e.ZKProofs != nil && request.RequireZKProof {
		proof, err := e.ZKProofs.GenerateProof(request.WorkloadID, teeMeasurement)
		if err != nil {
			return nil, fmt.Errorf("ZK proof generation failed: %w", err)
		}
		zkProof = proof
		log.Printf("ZK proof generated")
	}
	
	// Step 6: Run telemetry consensus verification
	var consensusResult *telemetry.ConsensusResult
	if e.TelemetryConsensus != nil {
		result, err := e.TelemetryConsensus.VerifyWorkloadCompletion(request.WorkloadID)
		if err != nil {
			return nil, fmt.Errorf("telemetry consensus verification failed: %w", err)
		}
		consensusResult = result
		log.Printf("Telemetry consensus completed: %s", result.ConsensusStatus)
	}
	
	// Calculate total processing time
	processingTime := time.Since(startTime)
	
	// Create response
	response := &ComputeResponse{
		WorkloadID:        request.WorkloadID,
		Success:           true,
		ProcessingTime:    processingTime,
		OCXQLQuery:        parsedQuery,
		TEEMeasurement:    teeMeasurement,
		USDPayment:        paymentResult,
		ZKProof:           zkProof,
		ConsensusResult:   consensusResult,
		Timestamp:         time.Now(),
	}
	
	log.Printf("Compute request processed successfully in %v", processingTime)
	return response, nil
}

// GetSystemStatus returns the status of all integrated systems
func (e *OCXProtocolEngine) GetSystemStatus() *SystemStatus {
	status := &SystemStatus{
		Timestamp: time.Now(),
		Components: make(map[string]ComponentStatus),
	}
	
	// OCX-QL status
	if e.OCXQL != nil {
		status.Components["ocxql"] = ComponentStatus{
			Enabled: true,
			Status:  "operational",
			Details: map[string]interface{}{
				"resource_types": e.OCXQL.GetResourceTypes(),
				"regions":        e.OCXQL.GetRegions(),
				"workloads":      e.OCXQL.GetWorkloadTypes(),
			},
		}
	}
	
	// Telemetry consensus status
	if e.TelemetryConsensus != nil {
		networkStatus := e.TelemetryConsensus.GetNetworkStatus()
		status.Components["telemetry_consensus"] = ComponentStatus{
			Enabled: true,
			Status:  "operational",
			Details: map[string]interface{}{
				"total_nodes":         networkStatus.TotalNodes,
				"byzantine_nodes":     len(networkStatus.ByzantineNodes),
				"byzantine_percentage": networkStatus.ByzantinePercentage,
				"block_height":        networkStatus.BlockHeight,
			},
		}
	}
	
	// TEE attestation status
	if e.TEEAttestation != nil {
		status.Components["tee_attestation"] = ComponentStatus{
			Enabled: true,
			Status:  "operational",
			Details: map[string]interface{}{
				"supported_tee_types": e.TEEAttestation.GetSupportedTEETypes(),
				"attestations_generated": e.TEEAttestation.GetAttestationCount(),
			},
		}
	}
	
	// USD tokenomics status
	if e.USDTokenomics != nil {
		stats := e.USDTokenomics.GetTokenomicsStats()
		status.Components["usd_tokenomics"] = ComponentStatus{
			Enabled: true,
			Status:  "operational",
			Details: map[string]interface{}{
				"total_verifiers":     stats.TotalVerifiers,
				"total_staked_usd":    stats.TotalStakedUSD,
				"transaction_fee_bps": stats.TransactionFeeBPS,
			},
		}
	}
	
	// ZK proofs status
	if e.ZKProofs != nil {
		proofStats := e.ZKProofs.GetProofStats()
		status.Components["zk_proofs"] = ComponentStatus{
			Enabled: true,
			Status:  "operational",
			Details: proofStats,
		}
	}
	
	return status
}

// ComputeRequest represents a compute request
type ComputeRequest struct {
	WorkloadID            string                 `json:"workload_id"`
	ProviderID            string                 `json:"provider_id"`
	CustomerID            string                 `json:"customer_id"`
	ResourceSpec          map[string]interface{} `json:"resource_spec"`
	ExpectedDuration      float64                `json:"expected_duration_seconds"`
	SLARequirements       map[string]float64     `json:"sla_requirements"`
	OCXQLQuery           string                 `json:"ocxql_query,omitempty"`
	USDCost              float64                `json:"usd_cost"`
	RequireTEEAttestation bool                   `json:"require_tee_attestation"`
	RequireZKProof       bool                   `json:"require_zk_proof"`
}

// ComputeResponse represents the response to a compute request
type ComputeResponse struct {
	WorkloadID        string                 `json:"workload_id"`
	Success           bool                   `json:"success"`
	ProcessingTime    time.Duration          `json:"processing_time_ms"`
	OCXQLQuery        *ocxql.OCXQLQuery      `json:"ocxql_query,omitempty"`
	TEEMeasurement    *tee.ComputeMeasurement `json:"tee_measurement,omitempty"`
	USDPayment        *tokenomics.USDPaymentResult `json:"usd_payment,omitempty"`
	ZKProof           *zkproofs.ZKProof      `json:"zk_proof,omitempty"`
	ConsensusResult   *telemetry.ConsensusResult `json:"consensus_result,omitempty"`
	Timestamp         time.Time              `json:"timestamp"`
}

// SystemStatus represents the status of all systems
type SystemStatus struct {
	Timestamp  time.Time                    `json:"timestamp"`
	Components map[string]ComponentStatus   `json:"components"`
}

// ComponentStatus represents the status of a single component
type ComponentStatus struct {
	Enabled bool                   `json:"enabled"`
	Status  string                 `json:"status"`
	Details map[string]interface{} `json:"details"`
}
