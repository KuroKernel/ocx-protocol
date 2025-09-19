// simple-test/main.go — Simple HTTP test
package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"
)

func main() {
	log.Println("🚀 Starting Simple Test Server...")
	
	mux := http.NewServeMux()
	
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("🏥 Health check from %s", r.RemoteAddr)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":    "ok",
			"timestamp": time.Now().Format(time.RFC3339),
			"version":   "Simple Test v1.0",
		})
	})

	port := "8083"
	log.Printf("🌐 Starting server on port %s", port)
	log.Printf("🏥 Health check: http://localhost:%s/health", port)
	
	// Add a timeout to see if it starts
	go func() {
		time.Sleep(2 * time.Second)
		log.Println("✅ Server should be running now")
	}()
	
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("❌ Server failed: %v", err)
	}
}
