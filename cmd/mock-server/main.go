// mock-server/main.go — Mock Server for testing
package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"
)

func main() {
	log.Println("🚀 Starting Mock Server...")
	
	mux := http.NewServeMux()
	
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("🏥 Health check from %s", r.RemoteAddr)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":    "ok",
			"timestamp": time.Now().Format(time.RFC3339),
			"version":   "Mock Server v1.0",
		})
	})

	mux.HandleFunc("/api/v1/execute", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("⚡ Execute request from %s", r.RemoteAddr)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "success",
			"result": "mock_execution_result",
			"cycles_used": 1000,
		})
	})

	mux.HandleFunc("/api/v1/verify", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("🔍 Verify request from %s", r.RemoteAddr)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"valid": true,
			"status": "success",
			"message": "Receipt verification completed",
		})
	})

	mux.HandleFunc("/api/v1/receipts", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("📋 Receipts request from %s", r.RemoteAddr)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"receipts": []map[string]interface{}{
				{
					"receipt_hash": "abc123",
					"lease_id":     "lease-1",
					"cycles_used":  1000,
					"created_at":   time.Now().Format(time.RFC3339),
				},
			},
			"count": 1,
			"status": "success",
		})
	})

	port := "8084"
	log.Printf("🌐 Starting mock server on port %s", port)
	log.Printf("🏥 Health check: http://localhost:%s/health", port)
	
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("❌ Server failed: %v", err)
	}
}