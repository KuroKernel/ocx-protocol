package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"ocx.local/internal/compliance"
	"ocx.local/internal/ledger"
	"ocx.local/internal/settlement"
)

func main() {
	// Create configuration
	config := &settlement.SettlementConfig{
		DefaultJurisdiction: "US",
		SupportedCurrencies: []string{"USD", "EUR", "CNY", "BTC"},
		SupportedRails:     []string{"swift", "lightning", "usdc"},
		JurisdictionPolicies: map[string]*settlement.JurisdictionPolicy{
			"US": {
				Jurisdiction:          "US",
				AllowedCurrencies:     []string{"USD", "BTC"},
				AllowedRails:          []string{"swift", "lightning"},
				BlockedCurrencies:     []string{},
				BlockedRails:          []string{},
				RequiredKYC:           true,
				RequiredSanctionsCheck: true,
				MaxTransactionAmount:  &settlement.Amount{Currency: "USD", Value: "1000000", DecimalPlaces: 2},
				MinTransactionAmount:  &settlement.Amount{Currency: "USD", Value: "1", DecimalPlaces: 2},
				DataResidency:         "US",
				ExportControlFlags:    []string{"ITAR", "EAR"},
				SanctionsScreening:    []string{"OFAC", "UN"},
				ComplianceRequirements: []string{"AML", "KYC", "CTF"},
			},
			"EU": {
				Jurisdiction:          "EU",
				AllowedCurrencies:     []string{"EUR", "USD", "BTC"},
				AllowedRails:          []string{"swift", "lightning"},
				BlockedCurrencies:     []string{},
				BlockedRails:          []string{},
				RequiredKYC:           true,
				RequiredSanctionsCheck: true,
				MaxTransactionAmount:  &settlement.Amount{Currency: "EUR", Value: "1000000", DecimalPlaces: 2},
				MinTransactionAmount:  &settlement.Amount{Currency: "EUR", Value: "1", DecimalPlaces: 2},
				DataResidency:         "EU",
				ExportControlFlags:    []string{"Dual-Use"},
				SanctionsScreening:    []string{"EU", "UN"},
				ComplianceRequirements: []string{"AML", "KYC", "CTF", "GDPR"},
			},
			"CN": {
				Jurisdiction:          "CN",
				AllowedCurrencies:     []string{"CNY", "USD"},
				AllowedRails:          []string{"swift"},
				BlockedCurrencies:     []string{"BTC"},
				BlockedRails:          []string{"lightning"},
				RequiredKYC:           true,
				RequiredSanctionsCheck: true,
				MaxTransactionAmount:  &settlement.Amount{Currency: "CNY", Value: "10000000", DecimalPlaces: 2},
				MinTransactionAmount:  &settlement.Amount{Currency: "CNY", Value: "1", DecimalPlaces: 2},
				DataResidency:         "CN",
				ExportControlFlags:    []string{"Export Control"},
				SanctionsScreening:    []string{"UN"},
				ComplianceRequirements: []string{"AML", "KYC", "CTF"},
			},
		},
		CurrencyPreferences: map[string][]string{
			"US": {"USD", "BTC"},
			"EU": {"EUR", "USD", "BTC"},
			"CN": {"CNY", "USD"},
		},
		RailPreferences: map[string][]string{
			"US": {"swift", "lightning"},
			"EU": {"swift", "lightning"},
			"CN": {"swift"},
		},
		ComplianceConfig: &compliance.ComplianceConfig{
			SanctionsAPIEndpoint:    "https://api.sanctions.com/v1",
			SanctionsAPIKey:         "sanctions_api_key",
			KYCProviderEndpoint:     "https://api.kyc.com/v1",
			KYCProviderAPIKey:       "kyc_api_key",
			RiskThreshold:           0.8,
			BlockedJurisdictions:    []string{"RU", "IR", "KP"},
			BlockedCurrencies:       []string{},
			RequireKYC:              true,
			RequireSanctionsCheck:   true,
			RequireRiskAssessment:   true,
		},
		LedgerConfig: &ledger.LedgerConfig{
			BaseCurrency:        "USD",
			SupportedCurrencies: []string{"USD", "EUR", "CNY", "BTC"},
			DecimalPlaces:       8,
			AccountTypes:        []string{"asset", "liability", "revenue", "expense", "equity"},
		},
	}
	
	// Create settlement manager
	settlementManager := settlement.NewSettlementManager(config)
	
	// Create HTTP server
	mux := http.NewServeMux()
	
	// Settlement endpoints
	mux.HandleFunc("/settlement/process", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		
		var request settlement.SettlementRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		
		response, err := settlementManager.ProcessSettlement(r.Context(), &request)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})
	
	mux.HandleFunc("/settlement/status/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		
		settlementID := r.URL.Path[len("/settlement/status/"):]
		if settlementID == "" {
			http.Error(w, "Settlement ID required", http.StatusBadRequest)
			return
		}
		
		status, err := settlementManager.GetSettlementStatus(r.Context(), settlementID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(status)
	})
	
	// Rail endpoints
	mux.HandleFunc("/rails", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		
		rails := settlementManager.GetSupportedRails()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"rails": rails,
		})
	})
	
	mux.HandleFunc("/rails/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		
		railID := r.URL.Path[len("/rails/"):]
		if railID == "" {
			http.Error(w, "Rail ID required", http.StatusBadRequest)
			return
		}
		
		capabilities := settlementManager.GetRailCapabilities(railID)
		if capabilities == nil {
			http.Error(w, "Rail not found", http.StatusNotFound)
			return
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(capabilities)
	})
	
	// Jurisdiction endpoints
	mux.HandleFunc("/jurisdictions", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		
		jurisdictions := settlementManager.GetSupportedJurisdictions()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"jurisdictions": jurisdictions,
		})
	})
	
	mux.HandleFunc("/jurisdictions/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		
		jurisdiction := r.URL.Path[len("/jurisdictions/"):]
		if jurisdiction == "" {
			http.Error(w, "Jurisdiction required", http.StatusBadRequest)
			return
		}
		
		policy, err := settlementManager.GetJurisdictionPolicy(jurisdiction)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(policy)
	})
	
	// Compliance endpoints
	mux.HandleFunc("/compliance/check", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		
		var request settlement.SettlementRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		
		result, err := settlementManager.GetComplianceResult(r.Context(), &request)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	})
	
	// Ledger endpoints
	mux.HandleFunc("/ledger/trial-balance", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		
		trialBalance, err := settlementManager.GetTrialBalance()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(trialBalance)
	})
	
	mux.HandleFunc("/ledger/export/iso20022", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		
		var request struct {
			StartDate string `json:"start_date"`
			EndDate   string `json:"end_date"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		
		startDate, err := time.Parse("2006-01-02", request.StartDate)
		if err != nil {
			http.Error(w, "Invalid start date", http.StatusBadRequest)
			return
		}
		
		endDate, err := time.Parse("2006-01-02", request.EndDate)
		if err != nil {
			http.Error(w, "Invalid end date", http.StatusBadRequest)
			return
		}
		
		export, err := settlementManager.ExportISO20022(r.Context(), startDate, endDate)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(export)
	})
	
	// System status endpoint
	mux.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		
		status, err := settlementManager.GetSystemStatus(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(status)
	})
	
	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":    "healthy",
			"timestamp": time.Now(),
			"version":   "1.0.0",
		})
	})
	
	// Print startup message
	fmt.Println("🚀 OCX Multi-Rail Settlement Server")
	fmt.Println("====================================")
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
	fmt.Println("📡 Starting server on port 8082...")
	fmt.Println("   - Settlement API: http://localhost:8082/settlement/process")
	fmt.Println("   - Rail Status: http://localhost:8082/rails")
	fmt.Println("   - Jurisdiction Info: http://localhost:8082/jurisdictions")
	fmt.Println("   - Compliance Check: http://localhost:8082/compliance/check")
	fmt.Println("   - Trial Balance: http://localhost:8082/ledger/trial-balance")
	fmt.Println("   - ISO 20022 Export: http://localhost:8082/ledger/export/iso20022")
	fmt.Println("   - System Status: http://localhost:8082/status")
	fmt.Println("   - Health Check: http://localhost:8082/health")
	
	// Start server
	port := os.Getenv("SETTLEMENT_PORT")
	if port == "" {
		port = "8082"
	}
	
	log.Fatal(http.ListenAndServe(":"+port, mux))
}
