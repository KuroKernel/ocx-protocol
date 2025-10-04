package v1_1

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// ReceiptManager manages the complete receipt lifecycle
type ReceiptManager struct {
	crypto    *CryptoManager
	replay    *ReplayProtection
	kms       *KMSManager
	siem      *SIEMExporter
	dashboard *DashboardManager
	db        *sql.DB
}

// NewReceiptManager creates a new receipt manager
func NewReceiptManager(db *sql.DB) (*ReceiptManager, error) {
	crypto, err := NewCryptoManager()
	if err != nil {
		return nil, fmt.Errorf("failed to create crypto manager: %w", err)
	}

	replay := NewReplayProtection(db, DefaultReplayRetention, DefaultClockSkew)
	kms := NewKMSManager()
	siem := NewSIEMExporter(db)
	dashboard := NewDashboardManager(db)

	// Register local KMS provider as default
	localKMS := NewLocalKMSProvider()
	kms.RegisterProvider("local", localKMS)
	kms.SetDefaultProvider("local")

	return &ReceiptManager{
		crypto:    crypto,
		replay:    replay,
		kms:       kms,
		siem:      siem,
		dashboard: dashboard,
		db:        db,
	}, nil
}

// Start starts all components of the receipt manager
func (rm *ReceiptManager) Start(ctx context.Context) error {
	// Start replay protection cleanup
	if err := rm.replay.StartCleanup(ctx); err != nil {
		return fmt.Errorf("failed to start replay protection: %w", err)
	}

	// Start dashboard updates
	if err := rm.dashboard.Start(ctx); err != nil {
		return fmt.Errorf("failed to start dashboard: %w", err)
	}

	return nil
}

// Stop stops all components of the receipt manager
func (rm *ReceiptManager) Stop() {
	rm.replay.StopCleanup()
	rm.dashboard.Stop()
}

// CreateReceipt creates a new receipt with full validation
func (rm *ReceiptManager) CreateReceipt(
	ctx context.Context,
	programHash, inputHash, outputHash [32]byte,
	gasUsed uint64,
	startedAt, finishedAt time.Time,
	issuerID string,
	keyID string,
	keyVersion uint32,
	hostCycles uint64,
	hostInfo map[string]string,
) (*ReceiptFull, error) {
	// Get the key pair from KMS
	provider, err := rm.kms.GetProvider("local")
	if err != nil {
		return nil, fmt.Errorf("failed to get KMS provider: %w", err)
	}

	keyPair, err := provider.GenerateKey(ctx, keyID, keyVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to get key pair: %w", err)
	}

	// Create the receipt
	receipt, err := rm.crypto.CreateReceipt(
		programHash, inputHash, outputHash,
		gasUsed, startedAt, finishedAt,
		issuerID, keyPair, hostCycles, hostInfo,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create receipt: %w", err)
	}

	// Check for replay attacks
	err = rm.replay.CheckAndRecordNonce(
		ctx, issuerID, receipt.Core.Nonce, time.Unix(0, int64(receipt.Core.IssuedAt)),
	)
	if err != nil {
		// Log the replay attack
		rm.dashboard.AddActivityEvent(ActivityEvent{
			Timestamp:   time.Now(),
			EventType:   "replay_attack",
			IssuerID:    issuerID,
			ReceiptID:   "",
			Description: fmt.Sprintf("Replay attack detected: %v", err),
			Success:     false,
		})
		return nil, fmt.Errorf("replay attack detected: %w", err)
	}

	// Store the receipt in the database
	err = rm.storeReceipt(ctx, receipt)
	if err != nil {
		return nil, fmt.Errorf("failed to store receipt: %w", err)
	}

	// Log successful receipt creation
	rm.dashboard.AddActivityEvent(ActivityEvent{
		Timestamp:   time.Now(),
		EventType:   "receipt_created",
		IssuerID:    issuerID,
		ReceiptID:   rm.generateReceiptID(receipt),
		Description: "Receipt created successfully",
		Success:     true,
	})

	return receipt, nil
}

// VerifyReceipt verifies a receipt with comprehensive validation
func (rm *ReceiptManager) VerifyReceipt(
	ctx context.Context,
	receipt *ReceiptFull,
	keyID string,
	keyVersion uint32,
) (*VerificationInfo, error) {
	// Get the public key from KMS
	provider, err := rm.kms.GetProvider("local")
	if err != nil {
		return nil, fmt.Errorf("failed to get KMS provider: %w", err)
	}

	publicKey, err := provider.GetPublicKey(ctx, keyID, keyVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to get public key: %w", err)
	}

	// Verify the receipt
	info, err := rm.crypto.ValidateReceipt(receipt, publicKey, DefaultClockSkew)
	if err != nil {
		// Log verification failure
		rm.dashboard.AddActivityEvent(ActivityEvent{
			Timestamp:   time.Now(),
			EventType:   "verification_failed",
			IssuerID:    receipt.Core.IssuerID,
			ReceiptID:   rm.generateReceiptID(receipt),
			Description: fmt.Sprintf("Verification failed: %v", err),
			Success:     false,
		})
		return info, fmt.Errorf("receipt verification failed: %w", err)
	}

	// Update verification status in database
	err = rm.updateVerificationStatus(ctx, rm.generateReceiptID(receipt), true)
	if err != nil {
		return info, fmt.Errorf("failed to update verification status: %w", err)
	}

	// Log successful verification
	rm.dashboard.AddActivityEvent(ActivityEvent{
		Timestamp:   time.Now(),
		EventType:   "verification_success",
		IssuerID:    receipt.Core.IssuerID,
		ReceiptID:   rm.generateReceiptID(receipt),
		Description: "Receipt verified successfully",
		Success:     true,
	})

	return info, nil
}

// storeReceipt stores a receipt in the database
func (rm *ReceiptManager) storeReceipt(ctx context.Context, receipt *ReceiptFull) error {
	receiptID := rm.generateReceiptID(receipt)

	query := `
		INSERT INTO ocx_receipts_v1_1 (
			id, issuer_id, key_version, program_hash, input_hash, output_hash,
			gas_used, started_at, finished_at, issued_at, nonce, float_mode,
			signature, host_cycles, host_info, verified
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16
		)
	`

	hostInfoJSON := "{}"
	if receipt.HostInfo != nil {
		// Convert host info to JSON
		hostInfoBytes, err := json.Marshal(receipt.HostInfo)
		if err == nil {
			hostInfoJSON = string(hostInfoBytes)
		}
	}

	_, err := rm.db.ExecContext(ctx, query,
		receiptID,
		receipt.Core.IssuerID,
		receipt.Core.KeyVersion,
		receipt.Core.ProgramHash[:],
		receipt.Core.InputHash[:],
		receipt.Core.OutputHash[:],
		receipt.Core.GasUsed,
		time.Unix(0, int64(receipt.Core.StartedAt)),
		time.Unix(0, int64(receipt.Core.FinishedAt)),
		time.Unix(0, int64(receipt.Core.IssuedAt)),
		receipt.Core.Nonce[:],
		receipt.Core.FloatMode,
		receipt.Signature[:],
		receipt.HostCycles,
		hostInfoJSON,
		false, // Initially unverified
	)

	return err
}

// updateVerificationStatus updates the verification status of a receipt
func (rm *ReceiptManager) updateVerificationStatus(ctx context.Context, receiptID string, verified bool) error {
	query := `UPDATE ocx_receipts_v1_1 SET verified = $1, updated_at = NOW() WHERE id = $2`
	_, err := rm.db.ExecContext(ctx, query, verified, receiptID)
	return err
}

// generateReceiptID generates a unique receipt ID
func (rm *ReceiptManager) generateReceiptID(receipt *ReceiptFull) string {
	// Use a combination of issuer ID, nonce, and issued timestamp
	data := fmt.Sprintf("%s:%x:%d",
		receipt.Core.IssuerID,
		receipt.Core.Nonce[:],
		receipt.Core.IssuedAt,
	)
	return fmt.Sprintf("receipt_%x", sha256.Sum256([]byte(data)))
}

// GetReceipt retrieves a receipt by ID
func (rm *ReceiptManager) GetReceipt(ctx context.Context, receiptID string) (*ReceiptFull, error) {
	query := `
		SELECT 
			issuer_id, key_version, program_hash, input_hash, output_hash,
			gas_used, started_at, finished_at, issued_at, nonce, float_mode,
			signature, host_cycles, host_info
		FROM ocx_receipts_v1_1
		WHERE id = $1
	`

	var issuerID, floatMode string
	var keyVersion uint32
	var gasUsed, hostCycles uint64
	var startedAt, finishedAt, issuedAt time.Time
	var programHash, inputHash, outputHash, nonce, signature []byte
	var hostInfoJSON string

	err := rm.db.QueryRowContext(ctx, query, receiptID).Scan(
		&issuerID, &keyVersion, &programHash, &inputHash, &outputHash,
		&gasUsed, &startedAt, &finishedAt, &issuedAt, &nonce, &floatMode,
		&signature, &hostCycles, &hostInfoJSON,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get receipt: %w", err)
	}

	// Parse host info
	var hostInfo map[string]string
	if hostInfoJSON != "" {
		if err := json.Unmarshal([]byte(hostInfoJSON), &hostInfo); err != nil {
			hostInfo = make(map[string]string)
		}
	}

	// Convert byte slices to fixed-size arrays
	var programHashArray, inputHashArray, outputHashArray [32]byte
	var nonceArray [16]byte
	var signatureArray [64]byte

	copy(programHashArray[:], programHash)
	copy(inputHashArray[:], inputHash)
	copy(outputHashArray[:], outputHash)
	copy(nonceArray[:], nonce)
	copy(signatureArray[:], signature)

	// Create the receipt
	receipt := &ReceiptFull{
		Core: ReceiptCore{
			ProgramHash: programHashArray,
			InputHash:   inputHashArray,
			OutputHash:  outputHashArray,
			GasUsed:     gasUsed,
			StartedAt:   uint64(startedAt.UnixNano()),
			FinishedAt:  uint64(finishedAt.UnixNano()),
			IssuerID:    issuerID,
			KeyVersion:  keyVersion,
			Nonce:       nonceArray,
			IssuedAt:    uint64(issuedAt.UnixNano()),
			FloatMode:   floatMode,
		},
		Signature:  signatureArray,
		HostCycles: hostCycles,
		HostInfo:   hostInfo,
	}

	return receipt, nil
}

// GetDashboard returns the dashboard manager
func (rm *ReceiptManager) GetDashboard() *DashboardManager {
	return rm.dashboard
}

// GetSIEMExporter returns the SIEM exporter
func (rm *ReceiptManager) GetSIEMExporter() *SIEMExporter {
	return rm.siem
}

// GetKMSManager returns the KMS manager
func (rm *ReceiptManager) GetKMSManager() *KMSManager {
	return rm.kms
}

// GetReplayProtection returns the replay protection manager
func (rm *ReceiptManager) GetReplayProtection() *ReplayProtection {
	return rm.replay
}
