package verify

import (
	"context"
	"crypto/ed25519"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fxamacker/cbor/v2"
	"ocx.local/pkg/receipt"
)

// BatchVerifier provides high-throughput receipt verification
// Optimized for 100+ receipts/second with parallel processing
type BatchVerifier struct {
	workers     int
	replayStore ReplayStore
	decMode     cbor.DecMode
}

// BatchVerifierConfig contains configuration for BatchVerifier
type BatchVerifierConfig struct {
	Workers     int         // Number of parallel workers (default: NumCPU)
	ReplayStore ReplayStore // Optional replay store for nonce checking
}

// BatchResult contains the result of verifying a single receipt
type BatchResult struct {
	Index   int           // Original index in batch
	Valid   bool          // Whether verification succeeded
	Error   error         // Error if verification failed
	Core    *receipt.ReceiptCore // Extracted core if valid
	Latency time.Duration // Time to verify this receipt
}

// BatchStats contains statistics about batch verification
type BatchStats struct {
	Total       int           // Total receipts processed
	Valid       int           // Number of valid receipts
	Invalid     int           // Number of invalid receipts
	TotalTime   time.Duration // Total processing time
	AvgLatency  time.Duration // Average per-receipt latency
	Throughput  float64       // Receipts per second
}

// NewBatchVerifier creates a new batch verifier
func NewBatchVerifier(cfg BatchVerifierConfig) (*BatchVerifier, error) {
	if cfg.Workers <= 0 {
		cfg.Workers = runtime.NumCPU()
	}

	decOpts := cbor.DecOptions{
		DupMapKey: cbor.DupMapKeyEnforcedAPF,
	}
	decMode, err := decOpts.DecMode()
	if err != nil {
		return nil, fmt.Errorf("failed to create CBOR decoder: %w", err)
	}

	return &BatchVerifier{
		workers:     cfg.Workers,
		replayStore: cfg.ReplayStore,
		decMode:     decMode,
	}, nil
}

// VerifyBatch verifies multiple receipts in parallel
// Returns results in the same order as input
func (bv *BatchVerifier) VerifyBatch(ctx context.Context, receipts []ReceiptBatch) ([]BatchResult, BatchStats) {
	startTime := time.Now()
	results := make([]BatchResult, len(receipts))

	if len(receipts) == 0 {
		return results, BatchStats{}
	}

	// Create work channel
	type workItem struct {
		index   int
		receipt ReceiptBatch
	}
	workCh := make(chan workItem, len(receipts))
	resultCh := make(chan BatchResult, len(receipts))

	// Spawn workers
	var wg sync.WaitGroup
	for i := 0; i < bv.workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for work := range workCh {
				select {
				case <-ctx.Done():
					resultCh <- BatchResult{
						Index: work.index,
						Valid: false,
						Error: ctx.Err(),
					}
				default:
					result := bv.verifyOne(work.receipt)
					result.Index = work.index
					resultCh <- result
				}
			}
		}()
	}

	// Send work
	go func() {
		for i, r := range receipts {
			workCh <- workItem{index: i, receipt: r}
		}
		close(workCh)
	}()

	// Collect results
	go func() {
		wg.Wait()
		close(resultCh)
	}()

	var validCount int32
	for result := range resultCh {
		results[result.Index] = result
		if result.Valid {
			atomic.AddInt32(&validCount, 1)
		}
	}

	totalTime := time.Since(startTime)
	stats := BatchStats{
		Total:      len(receipts),
		Valid:      int(validCount),
		Invalid:    len(receipts) - int(validCount),
		TotalTime:  totalTime,
		AvgLatency: totalTime / time.Duration(len(receipts)),
		Throughput: float64(len(receipts)) / totalTime.Seconds(),
	}

	return results, stats
}

// verifyOne verifies a single receipt
func (bv *BatchVerifier) verifyOne(batch ReceiptBatch) BatchResult {
	start := time.Now()

	var receiptFull receipt.ReceiptFull
	if err := bv.decMode.Unmarshal(batch.ReceiptData, &receiptFull); err != nil {
		return BatchResult{
			Valid:   false,
			Error:   fmt.Errorf("CBOR decode failed: %w", err),
			Latency: time.Since(start),
		}
	}

	// Validate public key
	if len(batch.PublicKey) != ed25519.PublicKeySize {
		return BatchResult{
			Valid:   false,
			Error:   fmt.Errorf("invalid public key length"),
			Latency: time.Since(start),
		}
	}

	// Validate signature
	if len(receiptFull.Signature) != ed25519.SignatureSize {
		return BatchResult{
			Valid:   false,
			Error:   fmt.Errorf("invalid signature length"),
			Latency: time.Since(start),
		}
	}

	// Canonicalize core
	coreBytes, err := receipt.CanonicalizeCore(&receiptFull.Core)
	if err != nil {
		return BatchResult{
			Valid:   false,
			Error:   fmt.Errorf("canonicalize failed: %w", err),
			Latency: time.Since(start),
		}
	}

	// Verify signature with domain separator
	msg := append([]byte("OCXv1|receipt|"), coreBytes...)
	if !ed25519.Verify(batch.PublicKey, msg, receiptFull.Signature) {
		return BatchResult{
			Valid:   false,
			Error:   fmt.Errorf("signature verification failed"),
			Latency: time.Since(start),
		}
	}

	// Check replay if store is configured
	if bv.replayStore != nil {
		isNew, err := bv.replayStore.CheckAndStore(receiptFull.Core.Nonce[:], receiptFull.Core.IssuedAt)
		if err != nil {
			return BatchResult{
				Valid:   false,
				Error:   fmt.Errorf("replay check error: %w", err),
				Latency: time.Since(start),
			}
		}
		if !isNew {
			return BatchResult{
				Valid:   false,
				Error:   fmt.Errorf("replay detected: nonce already seen"),
				Latency: time.Since(start),
			}
		}
	}

	// Validate invariants
	if err := validateCoreInvariants(&receiptFull.Core); err != nil {
		return BatchResult{
			Valid:   false,
			Error:   err,
			Latency: time.Since(start),
		}
	}

	return BatchResult{
		Valid:   true,
		Core:    &receiptFull.Core,
		Latency: time.Since(start),
	}
}

// validateCoreInvariants validates receipt core invariants
func validateCoreInvariants(core *receipt.ReceiptCore) error {
	if core.FinishedAt < core.StartedAt {
		return fmt.Errorf("finished_at must be >= started_at")
	}
	if core.GasUsed == 0 {
		return fmt.Errorf("gas_used must be > 0")
	}
	if core.IssuerID == "" {
		return fmt.Errorf("issuer_id must not be empty")
	}
	var zeroNonce [16]byte
	if core.Nonce == zeroNonce {
		return fmt.Errorf("nonce must not be zero")
	}
	if core.IssuedAt == 0 {
		return fmt.Errorf("issued_at must be set")
	}
	return nil
}

// VerifyBatchSimple is a convenience method that returns just validity bools
func (bv *BatchVerifier) VerifyBatchSimple(ctx context.Context, receipts []ReceiptBatch) ([]bool, error) {
	results, _ := bv.VerifyBatch(ctx, receipts)
	valid := make([]bool, len(results))
	for i, r := range results {
		valid[i] = r.Valid
	}
	return valid, nil
}

// StreamVerify verifies receipts from a channel, allowing for streaming verification
func (bv *BatchVerifier) StreamVerify(ctx context.Context, in <-chan ReceiptBatch, out chan<- BatchResult) {
	var wg sync.WaitGroup

	// Spawn workers
	for i := 0; i < bv.workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for batch := range in {
				select {
				case <-ctx.Done():
					return
				default:
					result := bv.verifyOne(batch)
					out <- result
				}
			}
		}()
	}

	wg.Wait()
	close(out)
}
