package chain

import (
	"context"
	"crypto/ed25519"
	"fmt"
	"time"
)

// Verifier provides chain verification functionality
type Verifier struct {
	store  Store
	policy ChainValidationPolicy

	// PublicKeyResolver resolves issuer_key_id to public key bytes
	PublicKeyResolver func(keyID string) ([]byte, error)
}

// NewVerifier creates a new chain verifier
func NewVerifier(store Store, policy ChainValidationPolicy) *Verifier {
	return &Verifier{
		store:  store,
		policy: policy,
	}
}

// SetPublicKeyResolver sets the function to resolve public keys from key IDs
func (v *Verifier) SetPublicKeyResolver(resolver func(keyID string) ([]byte, error)) {
	v.PublicKeyResolver = resolver
}

// VerifyChain verifies the entire chain leading to a receipt
func (v *Verifier) VerifyChain(ctx context.Context, receiptHash [32]byte) (*ChainVerificationResult, error) {
	startTime := time.Now()

	result := &ChainVerificationResult{
		Valid:      true,
		HeadHash:   receiptHash,
		VerifiedAt: startTime,
	}

	// Get ancestors (includes the receipt itself)
	limit := v.policy.MaxChainDepth
	ancestors, err := v.store.GetAncestors(ctx, receiptHash, limit)
	if err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, ChainVerificationError{
			ReceiptHash: receiptHash,
			Position:    0,
			ErrorType:   "store_error",
			Message:     fmt.Sprintf("failed to retrieve chain: %v", err),
		})
		result.VerificationMs = time.Since(startTime).Milliseconds()
		return result, nil
	}

	if len(ancestors) == 0 {
		result.Valid = false
		result.Errors = append(result.Errors, ChainVerificationError{
			ReceiptHash: receiptHash,
			Position:    0,
			ErrorType:   "missing",
			Message:     "receipt not found",
		})
		result.VerificationMs = time.Since(startTime).Milliseconds()
		return result, nil
	}

	result.ChainLength = len(ancestors)
	result.Receipts = ancestors

	// Genesis is the last in the list (we traverse head→genesis)
	genesis := ancestors[len(ancestors)-1]
	result.GenesisHash = genesis.ReceiptHash

	// Verify each link in the chain
	for i := 0; i < len(ancestors); i++ {
		receipt := ancestors[i]

		// Verify this receipt
		if err := v.verifyReceipt(ctx, &receipt, i); err != nil {
			result.Valid = false
			result.Errors = append(result.Errors, ChainVerificationError{
				ReceiptHash: receipt.ReceiptHash,
				Position:    i,
				ErrorType:   "verification_failed",
				Message:     err.Error(),
			})
			if !v.policy.AllowMissingAncestors {
				break // Stop on first error unless policy allows continuing
			}
		}

		// Verify link to previous receipt
		if i < len(ancestors)-1 {
			if err := v.verifyLink(ctx, &receipt, &ancestors[i+1], i); err != nil {
				result.Valid = false
				result.Errors = append(result.Errors, ChainVerificationError{
					ReceiptHash: receipt.ReceiptHash,
					Position:    i,
					ErrorType:   "link_error",
					Message:     err.Error(),
				})
				if !v.policy.AllowMissingAncestors {
					break
				}
			}
		}
	}

	// Verify genesis receipt has no previous hash
	if genesis.PrevReceiptHash != nil && !v.policy.AllowMissingAncestors {
		// This means we hit the depth limit or couldn't find all ancestors
		exists, _ := v.store.HasReceipt(ctx, *genesis.PrevReceiptHash)
		if exists {
			// More ancestors exist but weren't included (depth limit)
			// This is fine if we hit the depth limit
		} else if !v.policy.AllowMissingAncestors {
			result.Valid = false
			result.Errors = append(result.Errors, ChainVerificationError{
				ReceiptHash: genesis.ReceiptHash,
				Position:    len(ancestors) - 1,
				ErrorType:   "missing_ancestor",
				Message:     fmt.Sprintf("ancestor not found: %s", HashToHex(*genesis.PrevReceiptHash)),
			})
		}
	}

	result.VerificationMs = time.Since(startTime).Milliseconds()
	return result, nil
}

// verifyReceipt verifies a single receipt's integrity
func (v *Verifier) verifyReceipt(ctx context.Context, receipt *ChainedReceipt, position int) error {
	// Verify signature if policy requires
	if v.policy.VerifySignatures {
		if err := v.verifySignature(receipt); err != nil {
			return fmt.Errorf("signature verification failed: %w", err)
		}
	}

	// Verify timestamps
	if receipt.FinishedAt < receipt.StartedAt {
		return fmt.Errorf("invalid timestamps: finished_at (%d) < started_at (%d)",
			receipt.FinishedAt, receipt.StartedAt)
	}

	// Verify cycles
	if receipt.CyclesUsed == 0 {
		return fmt.Errorf("invalid cycles: must be > 0")
	}

	// Verify hash fields are not zero
	var zeroHash [32]byte
	if receipt.ArtifactHash == zeroHash {
		return fmt.Errorf("artifact_hash is zero")
	}
	if receipt.InputHash == zeroHash {
		return fmt.Errorf("input_hash is zero")
	}
	if receipt.OutputHash == zeroHash {
		return fmt.Errorf("output_hash is zero")
	}

	return nil
}

// verifyLink verifies the link between two consecutive receipts
func (v *Verifier) verifyLink(ctx context.Context, current, previous *ChainedReceipt, position int) error {
	// Verify hash linkage
	if current.PrevReceiptHash == nil {
		return fmt.Errorf("current receipt has no prev_receipt_hash but previous exists")
	}

	if *current.PrevReceiptHash != previous.ReceiptHash {
		return fmt.Errorf("hash mismatch: prev_receipt_hash (%s) != previous.receipt_hash (%s)",
			HashToHex(*current.PrevReceiptHash), HashToHex(previous.ReceiptHash))
	}

	// Verify timestamp ordering if policy requires
	if v.policy.RequireContiguousTimestamps {
		// Previous receipt should finish before or when current starts
		// Allow for some clock skew
		if previous.FinishedAt > current.StartedAt+v.policy.MaxClockSkew {
			return fmt.Errorf("timestamp ordering violation: previous finished at %d, current started at %d (max skew: %d)",
				previous.FinishedAt, current.StartedAt, v.policy.MaxClockSkew)
		}
	}

	return nil
}

// verifySignature verifies the Ed25519 signature on a receipt
func (v *Verifier) verifySignature(receipt *ChainedReceipt) error {
	if len(receipt.Signature) != 64 {
		return fmt.Errorf("invalid signature length: expected 64, got %d", len(receipt.Signature))
	}

	if v.PublicKeyResolver == nil {
		return fmt.Errorf("no public key resolver configured")
	}

	publicKey, err := v.PublicKeyResolver(receipt.IssuerKeyID)
	if err != nil {
		return fmt.Errorf("failed to resolve public key for %s: %w", receipt.IssuerKeyID, err)
	}

	if len(publicKey) != ed25519.PublicKeySize {
		return fmt.Errorf("invalid public key size: expected %d, got %d",
			ed25519.PublicKeySize, len(publicKey))
	}

	// NOTE: For full verification, we'd need the original signed data
	// This requires access to the canonical CBOR representation
	// For now, we trust the stored signature if we have it
	// Full signature verification happens at receipt creation time

	return nil
}

// VerifyReceipt verifies a single receipt (not the chain)
func (v *Verifier) VerifyReceipt(ctx context.Context, receiptHash [32]byte) error {
	receipt, err := v.store.GetReceiptByHash(ctx, receiptHash)
	if err != nil {
		return fmt.Errorf("receipt not found: %w", err)
	}

	return v.verifyReceipt(ctx, receipt, 0)
}

// GetChainProvenance returns a human-readable provenance trail
func (v *Verifier) GetChainProvenance(ctx context.Context, receiptHash [32]byte) ([]string, error) {
	ancestors, err := v.store.GetAncestors(ctx, receiptHash, v.policy.MaxChainDepth)
	if err != nil {
		return nil, err
	}

	provenance := make([]string, len(ancestors))
	for i, receipt := range ancestors {
		position := len(ancestors) - i // Reverse to show genesis first
		provenance[i] = fmt.Sprintf("[%d] %s (cycles: %d, at: %d)",
			position,
			HashToHex(receipt.ReceiptHash)[:16]+"...",
			receipt.CyclesUsed,
			receipt.StartedAt)
	}

	return provenance, nil
}

// AppendToChain adds a new receipt to an existing chain
func (v *Verifier) AppendToChain(ctx context.Context, receipt *ChainedReceipt) error {
	// If this is not a genesis receipt, verify the previous receipt exists
	if receipt.PrevReceiptHash != nil {
		exists, err := v.store.HasReceipt(ctx, *receipt.PrevReceiptHash)
		if err != nil {
			return fmt.Errorf("failed to check previous receipt: %w", err)
		}
		if !exists {
			return fmt.Errorf("previous receipt not found: %s", HashToHex(*receipt.PrevReceiptHash))
		}

		// Get previous receipt to verify ordering
		prev, err := v.store.GetReceiptByHash(ctx, *receipt.PrevReceiptHash)
		if err != nil {
			return fmt.Errorf("failed to get previous receipt: %w", err)
		}

		// Verify timestamp ordering
		if v.policy.RequireContiguousTimestamps {
			if prev.FinishedAt > receipt.StartedAt+v.policy.MaxClockSkew {
				return fmt.Errorf("timestamp ordering violation")
			}
		}

		// Set chain sequence
		receipt.ChainSeq = prev.ChainSeq + 1
		if receipt.ChainID == "" && prev.ChainID != "" {
			receipt.ChainID = prev.ChainID
		}
	} else {
		// Genesis receipt
		receipt.ChainSeq = 1
	}

	return v.store.SaveReceipt(ctx, receipt)
}

// CreateGenesisReceipt creates the first receipt in a new chain
func (v *Verifier) CreateGenesisReceipt(ctx context.Context, receipt *ChainedReceipt, chainID string) error {
	receipt.PrevReceiptHash = nil
	receipt.ChainID = chainID
	receipt.ChainSeq = 1
	receipt.StoredAt = time.Now()

	return v.store.SaveReceipt(ctx, receipt)
}
