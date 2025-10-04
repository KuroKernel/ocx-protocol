package v1_1

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"time"
)

// SIEMExporter handles export of receipt data for SIEM systems
type SIEMExporter struct {
	db *sql.DB
}

// NewSIEMExporter creates a new SIEM exporter
func NewSIEMExporter(db *sql.DB) *SIEMExporter {
	return &SIEMExporter{db: db}
}

// SIEMEvent represents a single SIEM event
type SIEMEvent struct {
	Timestamp     time.Time              `json:"timestamp"`
	EventType     string                 `json:"event_type"`
	ReceiptID     string                 `json:"receipt_id"`
	IssuerID      string                 `json:"issuer_id"`
	KeyVersion    uint32                 `json:"key_version"`
	ProgramHash   string                 `json:"program_hash"`
	InputHash     string                 `json:"input_hash"`
	OutputHash    string                 `json:"output_hash"`
	GasUsed       uint64                 `json:"gas_used"`
	ExecutionTime time.Duration          `json:"execution_time_ms"`
	Verification  *VerificationInfo      `json:"verification"`
	HostInfo      map[string]string      `json:"host_info"`
	Metadata      map[string]interface{} `json:"metadata"`
}

// ExportJSONL exports receipt data in JSONL format
func (se *SIEMExporter) ExportJSONL(ctx context.Context, w io.Writer, startTime, endTime time.Time) error {
	query := `
		SELECT 
			r.id,
			r.issuer_id,
			r.key_version,
			r.program_hash,
			r.input_hash,
			r.output_hash,
			r.gas_used,
			r.started_at,
			r.finished_at,
			r.issued_at,
			r.verified,
			r.host_info,
			r.created_at
		FROM ocx_receipts r
		WHERE r.created_at >= $1 AND r.created_at <= $2
		ORDER BY r.created_at ASC
	`

	rows, err := se.db.QueryContext(ctx, query, startTime, endTime)
	if err != nil {
		return fmt.Errorf("failed to query receipts: %w", err)
	}
	defer rows.Close()

	encoder := json.NewEncoder(w)

	for rows.Next() {
		var receiptID, issuerID, programHash, inputHash, outputHash string
		var keyVersion uint32
		var gasUsed uint64
		var startedAt, finishedAt, issuedAt, createdAt time.Time
		var verified bool
		var hostInfoJSON string

		err := rows.Scan(
			&receiptID, &issuerID, &keyVersion, &programHash, &inputHash, &outputHash,
			&gasUsed, &startedAt, &finishedAt, &issuedAt, &verified, &hostInfoJSON, &createdAt,
		)
		if err != nil {
			return fmt.Errorf("failed to scan receipt: %w", err)
		}

		// Parse host info
		var hostInfo map[string]string
		if hostInfoJSON != "" {
			if err := json.Unmarshal([]byte(hostInfoJSON), &hostInfo); err != nil {
				hostInfo = map[string]string{"error": "failed to parse host_info"}
			}
		}

		// Calculate execution time
		executionTime := finishedAt.Sub(startedAt)

		// Create SIEM event
		event := SIEMEvent{
			Timestamp:     createdAt,
			EventType:     "ocx_receipt",
			ReceiptID:     receiptID,
			IssuerID:      issuerID,
			KeyVersion:    keyVersion,
			ProgramHash:   programHash,
			InputHash:     inputHash,
			OutputHash:    outputHash,
			GasUsed:       gasUsed,
			ExecutionTime: executionTime,
			Verification: &VerificationInfo{
				IssuerID:       issuerID,
				KeyVersion:     keyVersion,
				SignatureValid: verified,
			},
			HostInfo: hostInfo,
			Metadata: map[string]interface{}{
				"started_at":  startedAt,
				"finished_at": finishedAt,
				"issued_at":   issuedAt,
			},
		}

		// Write as JSONL
		if err := encoder.Encode(event); err != nil {
			return fmt.Errorf("failed to encode event: %w", err)
		}
	}

	return rows.Err()
}

// ExportSplunkHEC exports data in Splunk HEC format
func (se *SIEMExporter) ExportSplunkHEC(ctx context.Context, w io.Writer, startTime, endTime time.Time) error {
	query := `
		SELECT 
			r.id,
			r.issuer_id,
			r.key_version,
			r.program_hash,
			r.input_hash,
			r.output_hash,
			r.gas_used,
			r.started_at,
			r.finished_at,
			r.issued_at,
			r.verified,
			r.host_info,
			r.created_at
		FROM ocx_receipts r
		WHERE r.created_at >= $1 AND r.created_at <= $2
		ORDER BY r.created_at ASC
	`

	rows, err := se.db.QueryContext(ctx, query, startTime, endTime)
	if err != nil {
		return fmt.Errorf("failed to query receipts: %w", err)
	}
	defer rows.Close()

	encoder := json.NewEncoder(w)

	for rows.Next() {
		var receiptID, issuerID, programHash, inputHash, outputHash string
		var keyVersion uint32
		var gasUsed uint64
		var startedAt, finishedAt, issuedAt, createdAt time.Time
		var verified bool
		var hostInfoJSON string

		err := rows.Scan(
			&receiptID, &issuerID, &keyVersion, &programHash, &inputHash, &outputHash,
			&gasUsed, &startedAt, &finishedAt, &issuedAt, &verified, &hostInfoJSON, &createdAt,
		)
		if err != nil {
			return fmt.Errorf("failed to scan receipt: %w", err)
		}

		// Parse host info
		var hostInfo map[string]string
		if hostInfoJSON != "" {
			if err := json.Unmarshal([]byte(hostInfoJSON), &hostInfo); err != nil {
				hostInfo = map[string]string{"error": "failed to parse host_info"}
			}
		}

		// Calculate execution time
		executionTime := finishedAt.Sub(startedAt)

		// Create Splunk HEC event
		hecEvent := map[string]interface{}{
			"time":       createdAt.Unix(),
			"host":       hostInfo["hostname"],
			"source":     "ocx-protocol",
			"sourcetype": "ocx:receipt",
			"index":      "ocx_receipts",
			"event": map[string]interface{}{
				"receipt_id":        receiptID,
				"issuer_id":         issuerID,
				"key_version":       keyVersion,
				"program_hash":      programHash,
				"input_hash":        inputHash,
				"output_hash":       outputHash,
				"gas_used":          gasUsed,
				"execution_time_ms": executionTime.Milliseconds(),
				"verified":          verified,
				"started_at":        startedAt,
				"finished_at":       finishedAt,
				"issued_at":         issuedAt,
				"host_info":         hostInfo,
			},
		}

		// Write as JSONL
		if err := encoder.Encode(hecEvent); err != nil {
			return fmt.Errorf("failed to encode HEC event: %w", err)
		}
	}

	return rows.Err()
}

// ExportAuditLog exports audit log data
func (se *SIEMExporter) ExportAuditLog(ctx context.Context, w io.Writer, startTime, endTime time.Time) error {
	query := `
		SELECT 
			al.id,
			al.timestamp,
			al.event_type,
			al.issuer_id,
			al.receipt_id,
			al.action,
			al.result,
			al.details,
			al.ip_address,
			al.user_agent
		FROM ocx_audit_log al
		WHERE al.timestamp >= $1 AND al.timestamp <= $2
		ORDER BY al.timestamp ASC
	`

	rows, err := se.db.QueryContext(ctx, query, startTime, endTime)
	if err != nil {
		return fmt.Errorf("failed to query audit log: %w", err)
	}
	defer rows.Close()

	encoder := json.NewEncoder(w)

	for rows.Next() {
		var id, eventType, issuerID, receiptID, action, result, details, ipAddress, userAgent string
		var timestamp time.Time

		err := rows.Scan(
			&id, &timestamp, &eventType, &issuerID, &receiptID,
			&action, &result, &details, &ipAddress, &userAgent,
		)
		if err != nil {
			return fmt.Errorf("failed to scan audit log entry: %w", err)
		}

		// Create audit event
		event := map[string]interface{}{
			"timestamp":  timestamp,
			"audit_id":   id,
			"event_type": eventType,
			"issuer_id":  issuerID,
			"receipt_id": receiptID,
			"action":     action,
			"result":     result,
			"details":    details,
			"ip_address": ipAddress,
			"user_agent": userAgent,
		}

		// Write as JSONL
		if err := encoder.Encode(event); err != nil {
			return fmt.Errorf("failed to encode audit event: %w", err)
		}
	}

	return rows.Err()
}

// GetExportStats returns statistics about exportable data
func (se *SIEMExporter) GetExportStats(ctx context.Context) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Count receipts
	var totalReceipts int
	err := se.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM ocx_receipts").Scan(&totalReceipts)
	if err != nil {
		return nil, err
	}
	stats["total_receipts"] = totalReceipts

	// Count audit log entries
	var totalAuditEntries int
	err = se.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM ocx_audit_log").Scan(&totalAuditEntries)
	if err != nil {
		return nil, err
	}
	stats["total_audit_entries"] = totalAuditEntries

	// Get date range
	var oldestReceipt, newestReceipt time.Time
	err = se.db.QueryRowContext(ctx, `
		SELECT MIN(created_at), MAX(created_at) 
		FROM ocx_receipts
	`).Scan(&oldestReceipt, &newestReceipt)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	if !oldestReceipt.IsZero() {
		stats["oldest_receipt"] = oldestReceipt
		stats["newest_receipt"] = newestReceipt
	}

	// Get verification stats
	var verifiedCount, unverifiedCount int
	err = se.db.QueryRowContext(ctx, `
		SELECT 
			COUNT(*) FILTER (WHERE verified = true),
			COUNT(*) FILTER (WHERE verified = false)
		FROM ocx_receipts
	`).Scan(&verifiedCount, &unverifiedCount)
	if err != nil {
		return nil, err
	}

	stats["verified_receipts"] = verifiedCount
	stats["unverified_receipts"] = unverifiedCount

	return stats, nil
}
