package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"ocx.local/pkg/receipt"
	"ocx.local/pkg/verify"
)

// WebSocketHandler handles real-time bidirectional verification
type WebSocketHandler struct {
	batchVerifier *verify.BatchVerifier
	compressor    *receipt.Compressor
	upgrader      websocket.Upgrader
	clients       sync.Map
	clientCount   int64
	metrics       *WSMetrics
}

// WSMetrics tracks WebSocket performance
type WSMetrics struct {
	TotalConnections  int64
	ActiveConnections int64
	MessagesReceived  int64
	MessagesSent      int64
	ReceiptsVerified  int64
	Errors            int64
}

// WSMessage represents a WebSocket message
type WSMessage struct {
	Type string          `json:"type"` // "verify", "batch_verify", "ping", "subscribe"
	ID   string          `json:"id"`   // Request ID for correlation
	Data json.RawMessage `json:"data"`
}

// WSResponse represents a WebSocket response
type WSResponse struct {
	Type      string          `json:"type"`      // "result", "error", "pong", "subscribed"
	ID        string          `json:"id"`        // Correlates to request ID
	Timestamp int64           `json:"timestamp"` // Unix nano
	Data      json.RawMessage `json:"data"`
}

// wsClient represents a connected WebSocket client
type wsClient struct {
	id         string
	conn       *websocket.Conn
	send       chan []byte
	handler    *WebSocketHandler
	ctx        context.Context
	cancel     context.CancelFunc
	subscribed bool
}

// NewWebSocketHandler creates a new WebSocket handler
func NewWebSocketHandler() (*WebSocketHandler, error) {
	bv, err := verify.NewBatchVerifier(verify.BatchVerifierConfig{})
	if err != nil {
		return nil, err
	}

	return &WebSocketHandler{
		batchVerifier: bv,
		compressor:    receipt.DefaultCompressor(),
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024 * 64,  // 64KB
			WriteBufferSize: 1024 * 64,  // 64KB
			CheckOrigin:     func(r *http.Request) bool { return true }, // Allow all origins
		},
		metrics: &WSMetrics{},
	}, nil
}

// ServeHTTP upgrades HTTP to WebSocket
func (h *WebSocketHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, "WebSocket upgrade failed", http.StatusBadRequest)
		return
	}

	clientID := fmt.Sprintf("ws-%d", time.Now().UnixNano())
	ctx, cancel := context.WithCancel(r.Context())

	client := &wsClient{
		id:      clientID,
		conn:    conn,
		send:    make(chan []byte, 256),
		handler: h,
		ctx:     ctx,
		cancel:  cancel,
	}

	h.clients.Store(clientID, client)
	atomic.AddInt64(&h.metrics.TotalConnections, 1)
	atomic.AddInt64(&h.metrics.ActiveConnections, 1)

	// Start read and write pumps
	go client.writePump()
	go client.readPump()

	// Send welcome message
	client.sendResponse(&WSResponse{
		Type:      "connected",
		ID:        clientID,
		Timestamp: time.Now().UnixNano(),
		Data:      json.RawMessage(`{"message":"Connected to OCX WebSocket"}`),
	})
}

// readPump reads messages from the WebSocket
func (c *wsClient) readPump() {
	defer func() {
		c.handler.clients.Delete(c.id)
		atomic.AddInt64(&c.handler.metrics.ActiveConnections, -1)
		c.cancel()
		c.conn.Close()
	}()

	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				atomic.AddInt64(&c.handler.metrics.Errors, 1)
			}
			return
		}

		atomic.AddInt64(&c.handler.metrics.MessagesReceived, 1)
		c.handleMessage(message)
	}
}

// writePump writes messages to the WebSocket
func (c *wsClient) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}
			atomic.AddInt64(&c.handler.metrics.MessagesSent, 1)

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}

		case <-c.ctx.Done():
			return
		}
	}
}

// handleMessage processes incoming messages
func (c *wsClient) handleMessage(data []byte) {
	var msg WSMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		c.sendError(msg.ID, "Invalid message format")
		return
	}

	switch msg.Type {
	case "verify":
		c.handleVerify(msg)
	case "batch_verify":
		c.handleBatchVerify(msg)
	case "ping":
		c.sendResponse(&WSResponse{
			Type:      "pong",
			ID:        msg.ID,
			Timestamp: time.Now().UnixNano(),
			Data:      json.RawMessage(`{}`),
		})
	case "subscribe":
		c.subscribed = true
		c.sendResponse(&WSResponse{
			Type:      "subscribed",
			ID:        msg.ID,
			Timestamp: time.Now().UnixNano(),
			Data:      json.RawMessage(`{"subscribed":true}`),
		})
	default:
		c.sendError(msg.ID, "Unknown message type: "+msg.Type)
	}
}

// handleVerify processes a single verification request
func (c *wsClient) handleVerify(msg WSMessage) {
	start := time.Now()

	var req struct {
		ReceiptData []byte `json:"receipt_data"`
		PublicKey   []byte `json:"public_key"`
	}

	if err := json.Unmarshal(msg.Data, &req); err != nil {
		c.sendError(msg.ID, "Invalid verify request")
		return
	}

	// Decompress if needed
	receiptData := req.ReceiptData
	if receipt.IsCompressed(receiptData) {
		decompressed, err := c.handler.compressor.DecompressReceipt(receiptData)
		if err != nil {
			c.sendError(msg.ID, "Decompression failed: "+err.Error())
			return
		}
		receiptData, _ = receipt.CanonicalizeFull(decompressed)
	}

	// Verify
	batch := verify.ReceiptBatch{
		ReceiptData: receiptData,
		PublicKey:   req.PublicKey,
	}

	results, _ := c.handler.batchVerifier.VerifyBatch(c.ctx, []verify.ReceiptBatch{batch})
	result := results[0]

	atomic.AddInt64(&c.handler.metrics.ReceiptsVerified, 1)

	// Build response
	resp := map[string]interface{}{
		"valid":      result.Valid,
		"latency_ms": float64(time.Since(start).Nanoseconds()) / 1e6,
	}

	if result.Error != nil {
		resp["error"] = result.Error.Error()
	}

	if result.Core != nil {
		resp["issuer_id"] = result.Core.IssuerID
		resp["gas_used"] = result.Core.GasUsed
	}

	respData, _ := json.Marshal(resp)
	c.sendResponse(&WSResponse{
		Type:      "result",
		ID:        msg.ID,
		Timestamp: time.Now().UnixNano(),
		Data:      respData,
	})
}

// handleBatchVerify processes batch verification request
func (c *wsClient) handleBatchVerify(msg WSMessage) {
	start := time.Now()

	var req struct {
		Receipts []struct {
			ReceiptData []byte `json:"receipt_data"`
			PublicKey   []byte `json:"public_key"`
		} `json:"receipts"`
	}

	if err := json.Unmarshal(msg.Data, &req); err != nil {
		c.sendError(msg.ID, "Invalid batch verify request")
		return
	}

	// Build batch
	batches := make([]verify.ReceiptBatch, len(req.Receipts))
	for i, r := range req.Receipts {
		receiptData := r.ReceiptData
		if receipt.IsCompressed(receiptData) {
			decompressed, err := c.handler.compressor.DecompressReceipt(receiptData)
			if err != nil {
				c.sendError(msg.ID, fmt.Sprintf("Decompression failed for receipt %d", i))
				return
			}
			receiptData, _ = receipt.CanonicalizeFull(decompressed)
		}
		batches[i] = verify.ReceiptBatch{
			ReceiptData: receiptData,
			PublicKey:   r.PublicKey,
		}
	}

	// Verify batch
	results, stats := c.handler.batchVerifier.VerifyBatch(c.ctx, batches)

	atomic.AddInt64(&c.handler.metrics.ReceiptsVerified, int64(len(batches)))

	// Build response
	resultsList := make([]map[string]interface{}, len(results))
	for i, r := range results {
		item := map[string]interface{}{
			"index": i,
			"valid": r.Valid,
		}
		if r.Error != nil {
			item["error"] = r.Error.Error()
		}
		resultsList[i] = item
	}

	resp := map[string]interface{}{
		"results":       resultsList,
		"total":         stats.Total,
		"valid":         stats.Valid,
		"invalid":       stats.Invalid,
		"throughput":    stats.Throughput,
		"total_time_ms": float64(stats.TotalTime.Nanoseconds()) / 1e6,
		"latency_ms":    float64(time.Since(start).Nanoseconds()) / 1e6,
	}

	respData, _ := json.Marshal(resp)
	c.sendResponse(&WSResponse{
		Type:      "batch_result",
		ID:        msg.ID,
		Timestamp: time.Now().UnixNano(),
		Data:      respData,
	})
}

// sendResponse sends a response to the client
func (c *wsClient) sendResponse(resp *WSResponse) {
	data, err := json.Marshal(resp)
	if err != nil {
		return
	}
	select {
	case c.send <- data:
	default:
		// Buffer full, drop message
	}
}

// sendError sends an error response
func (c *wsClient) sendError(id string, message string) {
	atomic.AddInt64(&c.handler.metrics.Errors, 1)
	errData, _ := json.Marshal(map[string]string{"error": message})
	c.sendResponse(&WSResponse{
		Type:      "error",
		ID:        id,
		Timestamp: time.Now().UnixNano(),
		Data:      errData,
	})
}

// Metrics returns current WebSocket metrics
func (h *WebSocketHandler) Metrics() WSMetrics {
	return WSMetrics{
		TotalConnections:  atomic.LoadInt64(&h.metrics.TotalConnections),
		ActiveConnections: atomic.LoadInt64(&h.metrics.ActiveConnections),
		MessagesReceived:  atomic.LoadInt64(&h.metrics.MessagesReceived),
		MessagesSent:      atomic.LoadInt64(&h.metrics.MessagesSent),
		ReceiptsVerified:  atomic.LoadInt64(&h.metrics.ReceiptsVerified),
		Errors:            atomic.LoadInt64(&h.metrics.Errors),
	}
}

// Broadcast sends a message to all connected clients
func (h *WebSocketHandler) Broadcast(resp *WSResponse) {
	data, err := json.Marshal(resp)
	if err != nil {
		return
	}

	h.clients.Range(func(key, value interface{}) bool {
		client := value.(*wsClient)
		if client.subscribed {
			select {
			case client.send <- data:
			default:
			}
		}
		return true
	})
}

// ActiveClients returns number of connected clients
func (h *WebSocketHandler) ActiveClients() int {
	return int(atomic.LoadInt64(&h.metrics.ActiveConnections))
}
