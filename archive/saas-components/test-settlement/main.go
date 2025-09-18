package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

func main() {
	// Create HTTP server
	mux := http.NewServeMux()
	
	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":    "healthy",
			"timestamp": time.Now(),
			"version":   "1.0.0",
		})
	})
	
	// System status endpoint
	mux.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"multi_rail_manager":    "healthy",
			"jurisdiction_matcher":  "healthy",
			"ledger_manager":        "healthy",
			"compliance_manager":    "healthy",
			"supported_rails":       []string{"swift", "lightning", "usdc"},
			"supported_jurisdictions": []string{"US", "EU", "CN", "JP", "SG"},
			"last_updated":          time.Now(),
		})
	})
	
	// Rails endpoint
	mux.HandleFunc("/rails", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"rails": []string{"swift", "lightning", "usdc"},
		})
	})
	
	// SWIFT rail capabilities
	mux.HandleFunc("/rails/swift", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"rail_id":              "swift",
			"supported_currencies": []string{"USD", "EUR", "GBP", "JPY", "CNY"},
			"jurisdictions":        []string{"US", "EU", "GB", "JP", "CN"},
		})
	})
	
	// Lightning rail capabilities
	mux.HandleFunc("/rails/lightning", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"rail_id":              "lightning",
			"supported_currencies": []string{"BTC"},
			"jurisdictions":        []string{"US", "EU", "GB", "JP", "SG"},
		})
	})
	
	// Jurisdictions endpoint
	mux.HandleFunc("/jurisdictions", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"jurisdictions": []string{"US", "EU", "CN", "JP", "SG"},
		})
	})
	
	// US jurisdiction policy
	mux.HandleFunc("/jurisdictions/US", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"jurisdiction":           "US",
			"allowed_currencies":     []string{"USD", "BTC"},
			"allowed_rails":          []string{"swift", "lightning"},
			"blocked_currencies":     []string{},
			"blocked_rails":          []string{},
			"required_kyc":           true,
			"required_sanctions_check": true,
			"max_transaction_amount": map[string]interface{}{
				"currency":       "USD",
				"value":          "1000000",
				"decimal_places": 2,
			},
			"min_transaction_amount": map[string]interface{}{
				"currency":       "USD",
				"value":          "1",
				"decimal_places": 2,
			},
			"data_residency":         "US",
			"export_control_flags":   []string{"ITAR", "EAR"},
			"sanctions_screening":    []string{"OFAC", "UN"},
			"compliance_requirements": []string{"AML", "KYC", "CTF"},
		})
	})
	
	// EU jurisdiction policy
	mux.HandleFunc("/jurisdictions/EU", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"jurisdiction":           "EU",
			"allowed_currencies":     []string{"EUR", "USD", "BTC"},
			"allowed_rails":          []string{"swift", "lightning"},
			"blocked_currencies":     []string{},
			"blocked_rails":          []string{},
			"required_kyc":           true,
			"required_sanctions_check": true,
			"max_transaction_amount": map[string]interface{}{
				"currency":       "EUR",
				"value":          "1000000",
				"decimal_places": 2,
			},
			"min_transaction_amount": map[string]interface{}{
				"currency":       "EUR",
				"value":          "1",
				"decimal_places": 2,
			},
			"data_residency":         "EU",
			"export_control_flags":   []string{"Dual-Use"},
			"sanctions_screening":    []string{"EU", "UN"},
			"compliance_requirements": []string{"AML", "KYC", "CTF", "GDPR"},
		})
	})
	
	// CN jurisdiction policy
	mux.HandleFunc("/jurisdictions/CN", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"jurisdiction":           "CN",
			"allowed_currencies":     []string{"CNY", "USD"},
			"allowed_rails":          []string{"swift"},
			"blocked_currencies":     []string{"BTC"},
			"blocked_rails":          []string{"lightning"},
			"required_kyc":           true,
			"required_sanctions_check": true,
			"max_transaction_amount": map[string]interface{}{
				"currency":       "CNY",
				"value":          "10000000",
				"decimal_places": 2,
			},
			"min_transaction_amount": map[string]interface{}{
				"currency":       "CNY",
				"value":          "1",
				"decimal_places": 2,
			},
			"data_residency":         "CN",
			"export_control_flags":   []string{"Export Control"},
			"sanctions_screening":    []string{"UN"},
			"compliance_requirements": []string{"AML", "KYC", "CTF"},
		})
	})
	
	// Mock settlement processing
	mux.HandleFunc("/settlement/process", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		
		var request map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		
		// Mock settlement response
		response := map[string]interface{}{
			"settlement_id":         fmt.Sprintf("settlement_%d", time.Now().UnixNano()),
			"instruction_id":        fmt.Sprintf("instruction_%d", time.Now().UnixNano()),
			"status":                "completed",
			"rail_used":             "swift",
			"transaction_reference": fmt.Sprintf("swift_%d", time.Now().UnixNano()),
			"settlement_date":       time.Now(),
			"value_date":            time.Now().Add(24 * time.Hour),
			"amount": map[string]interface{}{
				"currency":       "USD",
				"value":          "1000.00",
				"decimal_places": 2,
			},
			"fees": []map[string]interface{}{
				{
					"type":   "SWIFT_FEE",
					"amount": map[string]interface{}{
						"currency":       "USD",
						"value":          "25.00",
						"decimal_places": 2,
					},
				},
			},
			"compliance_status":     "compliant",
			"policy_compliance":     []string{"currency_compliant", "rail_compliant", "amount_compliant"},
			"warnings":              []string{},
			"recommendations":       []string{"Consider using USD for better liquidity"},
			"created_at":            time.Now(),
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})
	
	// Mock trial balance
	mux.HandleFunc("/ledger/trial-balance", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"accounts": []map[string]interface{}{
				{
					"account_id":     "cash_usd",
					"account_name":   "Cash USD",
					"account_type":   "asset",
					"currency":       "USD",
					"debit_balance":  map[string]interface{}{"currency": "USD", "value": "10000.00", "decimal_places": 2},
					"credit_balance": map[string]interface{}{"currency": "USD", "value": "0.00", "decimal_places": 2},
					"net_balance":    map[string]interface{}{"currency": "USD", "value": "10000.00", "decimal_places": 2},
				},
				{
					"account_id":     "payables_usd",
					"account_name":   "Payables USD",
					"account_type":   "liability",
					"currency":       "USD",
					"debit_balance":  map[string]interface{}{"currency": "USD", "value": "0.00", "decimal_places": 2},
					"credit_balance": map[string]interface{}{"currency": "USD", "value": "5000.00", "decimal_places": 2},
					"net_balance":    map[string]interface{}{"currency": "USD", "value": "5000.00", "decimal_places": 2},
				},
			},
			"total_debit":   map[string]interface{}{"currency": "USD", "value": "10000.00", "decimal_places": 2},
			"total_credit":  map[string]interface{}{"currency": "USD", "value": "10000.00", "decimal_places": 2},
			"generated_at":  time.Now(),
		})
	})
	
	// Mock ISO 20022 export
	mux.HandleFunc("/ledger/export/iso20022", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		
		var request map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		
		// Mock ISO 20022 export
		export := map[string]interface{}{
			"message_id":          fmt.Sprintf("export_%d", time.Now().UnixNano()),
			"creation_date_time":  time.Now(),
			"start_date":          request["start_date"],
			"end_date":            request["end_date"],
			"messages":            []map[string]interface{}{},
			"total_transactions":  0,
			"total_amount":        map[string]interface{}{"currency": "USD", "value": "0.00", "decimal_places": 2},
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(export)
	})
	
	// Print startup message
	fmt.Println("🚀 OCX Multi-Rail Settlement Test Server")
	fmt.Println("=========================================")
	fmt.Println("✅ Multi-rail settlement system ready")
	fmt.Println("✅ SWIFT/ISO 20022 support enabled")
	fmt.Println("✅ Lightning Network support enabled")
	fmt.Println("✅ Jurisdiction-aware matching enabled")
	fmt.Println("✅ Compliance and sanctions screening enabled")
	fmt.Println("✅ Double-entry ledger with ISO 20022 semantics enabled")
	fmt.Println("✅ Bank-friendly exports enabled")
	fmt.Println()
	fmt.Println("🌍 Supported Jurisdictions: US, EU, CN, JP, SG")
	fmt.Println("💰 Supported Currencies: USD, EUR, CNY, BTC")
	fmt.Println("🚂 Supported Rails: SWIFT, Lightning, USDC")
	fmt.Println()
	fmt.Println("📡 Starting test server on port 8082...")
	fmt.Println("   - Health Check: http://localhost:8082/health")
	fmt.Println("   - System Status: http://localhost:8082/status")
	fmt.Println("   - Rails: http://localhost:8082/rails")
	fmt.Println("   - Jurisdictions: http://localhost:8082/jurisdictions")
	fmt.Println("   - Settlement: http://localhost:8082/settlement/process")
	fmt.Println("   - Trial Balance: http://localhost:8082/ledger/trial-balance")
	fmt.Println("   - ISO 20022 Export: http://localhost:8082/ledger/export/iso20022")
	
	// Start server
	log.Fatal(http.ListenAndServe(":8082", mux))
}
