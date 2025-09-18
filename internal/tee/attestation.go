// internal/tee/attestation.go
package tee

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"time"
)

// TEEType represents the type of Trusted Execution Environment
type TEEType string

const (
	IntelSGX     TEEType = "intel_sgx"
	AMDSEV       TEEType = "amd_sev"
	ARMTrustZone TEEType = "arm_trustzone"
	AWSNitro     TEEType = "aws_nitro"
	GoogleAsylo  TEEType = "google_asylo"
)

// AttestationLevel represents the level of attestation
type AttestationLevel int

const (
	AttestationNone     AttestationLevel = 0
	AttestationBasic    AttestationLevel = 1 // Software attestation
	AttestationHardware AttestationLevel = 2 // TEE attestation
	AttestationCertified AttestationLevel = 3 // Third-party certified TEE
)

// TEEAttestation represents hardware-attested measurement from TEE
type TEEAttestation struct {
	AttestationID    string           `json:"attestation_id"`
	TEEType          TEEType          `json:"tee_type"`
	EnclaveHash      string           `json:"enclave_hash"`
	MeasurementData  []byte           `json:"measurement_data"`
	HardwareSignature []byte          `json:"hardware_signature"`
	Timestamp        time.Time        `json:"timestamp"`
	AttestationLevel AttestationLevel `json:"attestation_level"`
}

// ComputeMeasurement represents measurement taken inside TEE - tamper-proof
type ComputeMeasurement struct {
	MeasurementID        string           `json:"measurement_id"`
	WorkloadID           string           `json:"workload_id"`
	TEEAttestation       *TEEAttestation  `json:"tee_attestation"`
	
	// Actual performance metrics (measured in TEE)
	CPUCycles            int64            `json:"cpu_cycles"`
	MemoryOperations     int64            `json:"memory_operations"`
	GPUComputeUnits      int64            `json:"gpu_compute_units"`
	FloatingPointOps     int64            `json:"floating_point_ops"`
	MemoryBandwidthUsed  int64            `json:"memory_bandwidth_used"`
	
	// Timing measurements (tamper-proof timestamps)
	StartTimestampNS     int64            `json:"start_timestamp_ns"`
	EndTimestampNS       int64            `json:"end_timestamp_ns"`
	DurationNS           int64            `json:"duration_ns"`
	
	// Power and thermal (from hardware sensors)
	PowerConsumptionMW   int64            `json:"power_consumption_mw"`
	TemperatureMilliC    int64            `json:"temperature_millicelsius"`
	
	// Quality of service metrics
	MemoryErrors         int              `json:"memory_errors"`
	ThermalThrottlingEvents int           `json:"thermal_throttling_events"`
}

// ToAttestationBytes converts to format suitable for hardware attestation
func (cm *ComputeMeasurement) ToAttestationBytes() []byte {
	data := fmt.Sprintf("%s:%s:%d:%d:%d:%d:%d:%d:%d:%d:%d:%d:%d",
		cm.MeasurementID,
		cm.WorkloadID,
		cm.CPUCycles,
		cm.MemoryOperations,
		cm.GPUComputeUnits,
		cm.FloatingPointOps,
		cm.MemoryBandwidthUsed,
		cm.StartTimestampNS,
		cm.EndTimestampNS,
		cm.DurationNS,
		cm.PowerConsumptionMW,
		cm.TemperatureMilliC,
		cm.MemoryErrors,
		cm.ThermalThrottlingEvents,
	)
	return []byte(data)
}

// TEEMeasurementEngine simulates TEE-based measurement system
type TEEMeasurementEngine struct {
	TEEType          TEEType
	EnclaveHash      string
	AttestationCount int
}

// NewTEEMeasurementEngine creates a new TEE measurement engine
func NewTEEMeasurementEngine(teeType TEEType) *TEEMeasurementEngine {
	engine := &TEEMeasurementEngine{
		TEEType:          teeType,
		AttestationCount: 0,
	}
	
	engine.EnclaveHash = engine.generateEnclaveHash()
	return engine
}

// generateEnclaveHash generates hash of the measurement enclave code
func (tme *TEEMeasurementEngine) generateEnclaveHash() string {
	enclaveCode := fmt.Sprintf("TEE Measurement Enclave v1.0 - %s", tme.TEEType)
	hash := sha256.Sum256([]byte(enclaveCode))
	return hex.EncodeToString(hash[:])
}

// CreateMeasurement creates tamper-proof measurement inside TEE
func (tme *TEEMeasurementEngine) CreateMeasurement(workloadID string, simulatedComputeWork map[string]interface{}) (*ComputeMeasurement, error) {
	measurementID := fmt.Sprintf("measurement_%s_%d", workloadID, time.Now().Unix())
	
	// Simulate TEE measurement with realistic values
	baseTimeNS := time.Now().UnixNano()
	durationSeconds := getFloat64(simulatedComputeWork, "duration_seconds", 60)
	durationNS := int64(durationSeconds * 1_000_000_000)
	
	// Create the measurement
	measurement := &ComputeMeasurement{
		MeasurementID:        measurementID,
		WorkloadID:           workloadID,
		TEEAttestation:       nil, // Will be filled after attestation
		CPUCycles:            getInt64(simulatedComputeWork, "cpu_cycles", 1_000_000_000),
		MemoryOperations:     getInt64(simulatedComputeWork, "memory_ops", 50_000_000),
		GPUComputeUnits:      getInt64(simulatedComputeWork, "gpu_units", 4096),
		FloatingPointOps:     getInt64(simulatedComputeWork, "flops", 100_000_000_000),
		MemoryBandwidthUsed:  getInt64(simulatedComputeWork, "bandwidth", 500_000_000_000),
		StartTimestampNS:     baseTimeNS,
		EndTimestampNS:       baseTimeNS + durationNS,
		DurationNS:           durationNS,
		PowerConsumptionMW:   getInt64(simulatedComputeWork, "power_mw", 350_000),
		TemperatureMilliC:    getInt64(simulatedComputeWork, "temp_mc", 75_000),
		MemoryErrors:         0,
		ThermalThrottlingEvents: 0,
	}
	
	// Generate TEE attestation
	attestation := &TEEAttestation{
		AttestationID:    fmt.Sprintf("attestation_%s_%d", workloadID, time.Now().Unix()),
		TEEType:          tme.TEEType,
		EnclaveHash:      tme.EnclaveHash,
		MeasurementData:  measurement.ToAttestationBytes(),
		HardwareSignature: []byte("hardware_signed_attestation"),
		Timestamp:        time.Now(),
		AttestationLevel: AttestationHardware,
	}
	
	measurement.TEEAttestation = attestation
	tme.AttestationCount++
	
	log.Printf("Created TEE measurement for workload %s with %s attestation", workloadID, tme.TEEType)
	return measurement, nil
}

// VerifyAttestation verifies TEE attestation is valid
func (tme *TEEMeasurementEngine) VerifyAttestation(measurement *ComputeMeasurement) (bool, string) {
	attestation := measurement.TEEAttestation
	if attestation == nil {
		return false, "no attestation found"
	}
	
	// Verify attestation level
	if attestation.AttestationLevel < AttestationHardware {
		return false, "insufficient attestation level"
	}
	
	// Verify enclave hash matches
	if attestation.EnclaveHash != tme.EnclaveHash {
		return false, "enclave hash mismatch"
	}
	
	// Verify timestamp is recent
	if time.Since(attestation.Timestamp) > 5*time.Minute {
		return false, "attestation timestamp too old"
	}
	
	return true, "attestation verified successfully"
}

// GetSupportedTEETypes returns supported TEE types
func (tme *TEEMeasurementEngine) GetSupportedTEETypes() []string {
	return []string{
		string(IntelSGX),
		string(AMDSEV),
		string(ARMTrustZone),
		string(AWSNitro),
		string(GoogleAsylo),
	}
}

// GetAttestationCount returns the number of attestations generated
func (tme *TEEMeasurementEngine) GetAttestationCount() int {
	return tme.AttestationCount
}

// Helper functions
func getInt64(data map[string]interface{}, key string, defaultValue int64) int64 {
	if value, ok := data[key]; ok {
		if intVal, ok := value.(int64); ok {
			return intVal
		}
		if floatVal, ok := value.(float64); ok {
			return int64(floatVal)
		}
	}
	return defaultValue
}

func getFloat64(data map[string]interface{}, key string, defaultValue float64) float64 {
	if value, ok := data[key]; ok {
		if floatVal, ok := value.(float64); ok {
			return floatVal
		}
		if intVal, ok := value.(int64); ok {
			return float64(intVal)
		}
	}
	return defaultValue
}
