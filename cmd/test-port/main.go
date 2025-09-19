// test-port/main.go — Test server on different port
package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ocx.local/internal/version"
)

func main() {
	log.Println("🚀 Starting Test Server on port 9000...")
	
	mux := http.NewServeMux()
	
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("🏥 Health check from %s", r.RemoteAddr)
		w.Header().Set("Content-Type", "application/json")
		
		v := version.GetVersion()
		response := map[string]interface{}{
			"status":     "ok",
			"timestamp":  time.Now().Format(time.RFC3339),
			"version":    "OCX Protocol v1.0.0-rc.1",
			"spec_hash":  v.SpecHash,
			"build":      v.Build,
			"git_commit": v.GitCommit,
			"git_branch": v.GitBranch,
			"go_version": v.GoVersion,
			"platform":   v.Platform,
			"arch":       v.Arch,
		}
		
		json.NewEncoder(w).Encode(response)
	})

	mux.HandleFunc("/execute", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("⚡ Execute request from %s", r.RemoteAddr)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "success",
			"result": "mock_execution_result",
			"cycles_used": 1000,
		})
	})

	mux.HandleFunc("/api/v1/execute", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("⚡ API Execute request from %s", r.RemoteAddr)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "success",
			"result": "mock_execution_result",
			"cycles_used": 1000,
		})
	})

	mux.HandleFunc("/verify", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("🔍 Verify request from %s", r.RemoteAddr)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"valid": true,
			"status": "success",
			"message": "Receipt verification completed",
		})
	})

	mux.HandleFunc("/api/v1/verify", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("🔍 API Verify request from %s", r.RemoteAddr)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"valid": true,
			"status": "success",
			"message": "Receipt verification completed",
		})
	})

	mux.HandleFunc("/api/v1/receipts", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("📋 API Receipts request from %s", r.RemoteAddr)
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

	// Set up signal handling for graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	
	go func() {
		<-c
		log.Println("🛑 Shutting down server...")
		os.Exit(0)
	}()
	
	port := "9000"
	log.Printf("🌐 Starting server on port %s", port)
	log.Printf("🏥 Health check: http://localhost:%s/health", port)
	
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("❌ Server failed: %v", err)
	}
}
