package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"ocx.local/pkg/receipt"
	"ocx.local/pkg/verify"
)

// StreamingVerifier handles real-time receipt verification via SSE
type StreamingVerifier struct {
	batchVerifier *verify.BatchVerifier
	compressor    *receipt.Compressor
	clients       sync.Map // map[string]*streamClient
	clientCount   int64
	metrics       *StreamMetrics
}

// StreamMetrics tracks streaming performance
type StreamMetrics struct {
	TotalConnections   int64
	ActiveConnections  int64
	ReceiptsVerified   int64
	ReceiptsValid      int64
	ReceiptsInvalid    int64
	BytesReceived      int64
	BytesSent          int64
	AvgLatencyNs       int64
}

// streamClient represents a connected streaming client
type streamClient struct {
	id       string
	ctx      context.Context
	cancel   context.CancelFunc
	events   chan *StreamEvent
	created  time.Time
}

// StreamEvent is sent to clients via SSE
type StreamEvent struct {
	Type      string          `json:"type"`      // "result", "error", "heartbeat", "stats"
	ID        string          `json:"id"`        // Request ID
	Timestamp int64           `json:"timestamp"` // Unix nano
	Data      json.RawMessage `json:"data"`
}

// VerifyResult is the verification result sent to clients
type VerifyResult struct {
	RequestID   string        `json:"request_id"`
	Valid       bool          `json:"valid"`
	Error       string        `json:"error,omitempty"`
	ReceiptHash string        `json:"receipt_hash,omitempty"`
	IssuerID    string        `json:"issuer_id,omitempty"`
	GasUsed     uint64        `json:"gas_used,omitempty"`
	LatencyMs   float64       `json:"latency_ms"`
}

// StreamingVerifierConfig contains configuration
type StreamingVerifierConfig struct {
	MaxClients       int           // Max concurrent clients (default: 1000)
	HeartbeatInterval time.Duration // Heartbeat interval (default: 30s)
	EventBufferSize  int           // Per-client event buffer (default: 100)
}

// NewStreamingVerifier creates a new streaming verifier
func NewStreamingVerifier(cfg StreamingVerifierConfig) (*StreamingVerifier, error) {
	if cfg.MaxClients <= 0 {
		cfg.MaxClients = 1000
	}
	if cfg.HeartbeatInterval <= 0 {
		cfg.HeartbeatInterval = 30 * time.Second
	}
	if cfg.EventBufferSize <= 0 {
		cfg.EventBufferSize = 100
	}

	bv, err := verify.NewBatchVerifier(verify.BatchVerifierConfig{})
	if err != nil {
		return nil, fmt.Errorf("failed to create batch verifier: %w", err)
	}

	sv := &StreamingVerifier{
		batchVerifier: bv,
		compressor:    receipt.DefaultCompressor(),
		metrics:       &StreamMetrics{},
	}

	return sv, nil
}

// ServeHTTP handles streaming verification requests
func (sv *StreamingVerifier) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// SSE stream connection
		sv.handleSSE(w, r)
	case http.MethodPost:
		// Submit receipt for verification
		sv.handleVerify(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleSSE establishes a Server-Sent Events connection
func (sv *StreamingVerifier) handleSSE(w http.ResponseWriter, r *http.Request) {
	// Check if SSE is supported
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "SSE not supported", http.StatusInternalServerError)
		return
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // Disable nginx buffering

	// Create client
	clientID := fmt.Sprintf("client-%d", time.Now().UnixNano())
	ctx, cancel := context.WithCancel(r.Context())
	client := &streamClient{
		id:      clientID,
		ctx:     ctx,
		cancel:  cancel,
		events:  make(chan *StreamEvent, 100),
		created: time.Now(),
	}

	sv.clients.Store(clientID, client)
	atomic.AddInt64(&sv.metrics.TotalConnections, 1)
	atomic.AddInt64(&sv.metrics.ActiveConnections, 1)

	defer func() {
		sv.clients.Delete(clientID)
		atomic.AddInt64(&sv.metrics.ActiveConnections, -1)
		cancel()
		close(client.events)
	}()

	// Send initial connection event
	sv.sendEvent(w, flusher, &StreamEvent{
		Type:      "connected",
		ID:        clientID,
		Timestamp: time.Now().UnixNano(),
		Data:      json.RawMessage(`{"message":"Connected to OCX verification stream"}`),
	})

	// Heartbeat ticker
	heartbeat := time.NewTicker(30 * time.Second)
	defer heartbeat.Stop()

	// Event loop
	for {
		select {
		case <-ctx.Done():
			return
		case <-r.Context().Done():
			return
		case <-heartbeat.C:
			sv.sendEvent(w, flusher, &StreamEvent{
				Type:      "heartbeat",
				ID:        clientID,
				Timestamp: time.Now().UnixNano(),
				Data:      json.RawMessage(`{}`),
			})
		case event := <-client.events:
			sv.sendEvent(w, flusher, event)
		}
	}
}

// sendEvent writes an SSE event to the client
func (sv *StreamingVerifier) sendEvent(w http.ResponseWriter, flusher http.Flusher, event *StreamEvent) {
	data, err := json.Marshal(event)
	if err != nil {
		return
	}

	fmt.Fprintf(w, "event: %s\n", event.Type)
	fmt.Fprintf(w, "id: %s\n", event.ID)
	fmt.Fprintf(w, "data: %s\n\n", data)
	flusher.Flush()

	atomic.AddInt64(&sv.metrics.BytesSent, int64(len(data)+50))
}

// handleVerify handles receipt verification POST requests
func (sv *StreamingVerifier) handleVerify(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	// Parse request
	var req struct {
		ClientID    string `json:"client_id"`    // Optional: stream results to this client
		RequestID   string `json:"request_id"`   // For correlation
		ReceiptData []byte `json:"receipt_data"` // CBOR or compressed receipt
		PublicKey   []byte `json:"public_key"`   // Signer's public key
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	atomic.AddInt64(&sv.metrics.BytesReceived, int64(len(req.ReceiptData)))

	// Decompress if needed
	receiptData := req.ReceiptData
	if receipt.IsCompressed(receiptData) {
		decompressed, err := sv.compressor.DecompressReceipt(receiptData)
		if err != nil {
			sv.sendResult(w, req.ClientID, &VerifyResult{
				RequestID: req.RequestID,
				Valid:     false,
				Error:     "Decompression failed: " + err.Error(),
				LatencyMs: float64(time.Since(start).Nanoseconds()) / 1e6,
			})
			return
		}
		// Re-encode for verification
		var err2 error
		receiptData, err2 = receipt.CanonicalizeFull(decompressed)
		if err2 != nil {
			sv.sendResult(w, req.ClientID, &VerifyResult{
				RequestID: req.RequestID,
				Valid:     false,
				Error:     "Re-encode failed: " + err2.Error(),
				LatencyMs: float64(time.Since(start).Nanoseconds()) / 1e6,
			})
			return
		}
	}

	// Verify
	batch := verify.ReceiptBatch{
		ReceiptData: receiptData,
		PublicKey:   req.PublicKey,
	}

	results, _ := sv.batchVerifier.VerifyBatch(r.Context(), []verify.ReceiptBatch{batch})
	result := results[0]

	atomic.AddInt64(&sv.metrics.ReceiptsVerified, 1)
	if result.Valid {
		atomic.AddInt64(&sv.metrics.ReceiptsValid, 1)
	} else {
		atomic.AddInt64(&sv.metrics.ReceiptsInvalid, 1)
	}

	// Build response
	verifyResult := &VerifyResult{
		RequestID: req.RequestID,
		Valid:     result.Valid,
		LatencyMs: float64(time.Since(start).Nanoseconds()) / 1e6,
	}

	if result.Error != nil {
		verifyResult.Error = result.Error.Error()
	}

	if result.Core != nil {
		verifyResult.IssuerID = result.Core.IssuerID
		verifyResult.GasUsed = result.Core.GasUsed
	}

	sv.sendResult(w, req.ClientID, verifyResult)
}

// sendResult sends verification result via HTTP and optionally SSE
func (sv *StreamingVerifier) sendResult(w http.ResponseWriter, clientID string, result *VerifyResult) {
	// Always send HTTP response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)

	// Also stream to client if connected
	if clientID != "" {
		if clientVal, ok := sv.clients.Load(clientID); ok {
			client := clientVal.(*streamClient)
			data, _ := json.Marshal(result)
			select {
			case client.events <- &StreamEvent{
				Type:      "result",
				ID:        result.RequestID,
				Timestamp: time.Now().UnixNano(),
				Data:      data,
			}:
			default:
				// Buffer full, skip
			}
		}
	}
}

// Metrics returns current streaming metrics
func (sv *StreamingVerifier) Metrics() StreamMetrics {
	return StreamMetrics{
		TotalConnections:  atomic.LoadInt64(&sv.metrics.TotalConnections),
		ActiveConnections: atomic.LoadInt64(&sv.metrics.ActiveConnections),
		ReceiptsVerified:  atomic.LoadInt64(&sv.metrics.ReceiptsVerified),
		ReceiptsValid:     atomic.LoadInt64(&sv.metrics.ReceiptsValid),
		ReceiptsInvalid:   atomic.LoadInt64(&sv.metrics.ReceiptsInvalid),
		BytesReceived:     atomic.LoadInt64(&sv.metrics.BytesReceived),
		BytesSent:         atomic.LoadInt64(&sv.metrics.BytesSent),
	}
}

// BroadcastEvent sends an event to all connected clients
func (sv *StreamingVerifier) BroadcastEvent(event *StreamEvent) {
	sv.clients.Range(func(key, value interface{}) bool {
		client := value.(*streamClient)
		select {
		case client.events <- event:
		default:
			// Buffer full, skip
		}
		return true
	})
}

// ActiveClients returns the number of connected clients
func (sv *StreamingVerifier) ActiveClients() int {
	return int(atomic.LoadInt64(&sv.metrics.ActiveConnections))
}
