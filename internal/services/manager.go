package services

import (
	"context"

	"ocx.local/internal/config"
	"ocx.local/internal/kyc"
	"ocx.local/internal/legal"
	"ocx.local/internal/payments"
	"ocx.local/internal/support"
	"ocx.local/internal/verification"
)

// ServiceManager manages all external services
type ServiceManager struct {
	config *config.Config
	
	// Payment services
	StripeProcessor *payments.StripePaymentProcessor
	USDCProcessor   *payments.USDCProcessor
	
	// KYC services
	JumioProvider   *kyc.JumioKYCProvider
	
	// Verification services
	HardwareVerifier *verification.HardwareVerifier
	
	// Support services
	ZendeskManager  *support.ZendeskSupportManager
	
	// Legal services
	LegalManager    *legal.LegalManager
}

// NewServiceManager creates a new service manager
func NewServiceManager(cfg *config.Config) *ServiceManager {
	sm := &ServiceManager{
		config: cfg,
	}
	
	// Initialize payment services
	sm.StripeProcessor = payments.NewStripePaymentProcessor(&cfg.Stripe)
	sm.USDCProcessor = payments.NewUSDCProcessor(&cfg.USDC)
	
	// Initialize KYC services
	sm.JumioProvider = kyc.NewJumioKYCProvider(&cfg.Jumio)
	
	// Initialize verification services
	sm.HardwareVerifier = verification.NewHardwareVerifier(&cfg.Hardware)
	
	// Initialize support services
	sm.ZendeskManager = support.NewZendeskSupportManager(&cfg.Zendesk)
	
	// Initialize legal services
	sm.LegalManager = legal.NewLegalManager(&cfg.Legal)
	
	return sm
}

// HealthCheck checks the health of all services
func (sm *ServiceManager) HealthCheck(ctx context.Context) map[string]interface{} {
	health := make(map[string]interface{})
	
	// Check payment services
	health["stripe"] = map[string]interface{}{
		"configured": sm.StripeProcessor.IsConfigured(),
		"status":     "healthy",
	}
	
	health["usdc"] = map[string]interface{}{
		"configured": sm.USDCProcessor.IsConfigured(),
		"status":     "healthy",
	}
	
	// Check KYC services
	health["jumio"] = map[string]interface{}{
		"configured": sm.JumioProvider.IsConfigured(),
		"status":     "healthy",
	}
	
	// Check verification services
	health["hardware_verification"] = map[string]interface{}{
		"configured": sm.HardwareVerifier.IsConfigured(),
		"status":     "healthy",
	}
	
	// Check support services
	health["zendesk"] = map[string]interface{}{
		"configured": sm.ZendeskManager.IsConfigured(),
		"status":     "healthy",
	}
	
	// Check legal services
	health["legal"] = map[string]interface{}{
		"configured": sm.LegalManager.IsConfigured(),
		"status":     "healthy",
	}
	
	return health
}

// GetConfigurationStatus returns the configuration status of all services
func (sm *ServiceManager) GetConfigurationStatus() map[string]interface{} {
	status := make(map[string]interface{})
	
	// Payment services status
	status["payments"] = map[string]interface{}{
		"stripe": map[string]interface{}{
			"configured": sm.StripeProcessor.IsConfigured(),
			"secret_key": sm.config.Stripe.SecretKey != "",
			"publishable_key": sm.config.Stripe.PublishableKey != "",
		},
		"usdc": map[string]interface{}{
			"configured": sm.USDCProcessor.IsConfigured(),
			"rpc_url": sm.config.USDC.RPCURL != "",
			"contract_address": sm.config.USDC.ContractAddr != "",
			"private_key": sm.config.USDC.PrivateKey != "",
		},
	}
	
	// KYC services status
	status["kyc"] = map[string]interface{}{
		"jumio": map[string]interface{}{
			"configured": sm.JumioProvider.IsConfigured(),
			"api_key": sm.config.Jumio.APIKey != "",
			"api_secret": sm.config.Jumio.APISecret != "",
		},
	}
	
	// Verification services status
	status["verification"] = map[string]interface{}{
		"hardware": map[string]interface{}{
			"configured": sm.HardwareVerifier.IsConfigured(),
			"benchmark_timeout": sm.config.Hardware.BenchmarkTimeout,
			"min_score": sm.config.Hardware.MinScore,
		},
	}
	
	// Support services status
	status["support"] = map[string]interface{}{
		"zendesk": map[string]interface{}{
			"configured": sm.ZendeskManager.IsConfigured(),
			"domain": sm.config.Zendesk.Domain != "",
			"email": sm.config.Zendesk.Email != "",
			"api_token": sm.config.Zendesk.APIToken != "",
		},
	}
	
	// Legal services status
	status["legal"] = map[string]interface{}{
		"configured": sm.LegalManager.IsConfigured(),
		"terms_version": sm.config.Legal.TermsVersion,
		"privacy_version": sm.config.Legal.PrivacyVersion,
		"sla_version": sm.config.Legal.SLAVersion,
	}
	
	return status
}

// GetMissingConfiguration returns what needs to be configured
func (sm *ServiceManager) GetMissingConfiguration() []string {
	var missing []string
	
	// Check payment services
	if !sm.StripeProcessor.IsConfigured() {
		missing = append(missing, "Stripe: Secret key and publishable key required")
	}
	
	if !sm.USDCProcessor.IsConfigured() {
		missing = append(missing, "USDC: RPC URL, contract address, and private key required")
	}
	
	// Check KYC services
	if !sm.JumioProvider.IsConfigured() {
		missing = append(missing, "Jumio: API key and secret required")
	}
	
	// Check verification services
	if !sm.HardwareVerifier.IsConfigured() {
		missing = append(missing, "Hardware Verification: Benchmark timeout and min score required")
	}
	
	// Check support services
	if !sm.ZendeskManager.IsConfigured() {
		missing = append(missing, "Zendesk: Domain, email, and API token required")
	}
	
	// Check legal services
	if !sm.LegalManager.IsConfigured() {
		missing = append(missing, "Legal: Terms, privacy, and SLA versions required")
	}
	
	return missing
}

// GetAPIKeysNeeded returns the API keys that need to be configured
func (sm *ServiceManager) GetAPIKeysNeeded() map[string][]string {
	keys := make(map[string][]string)
	
	// Payment services
	if sm.config.Stripe.SecretKey == "" {
		keys["stripe"] = append(keys["stripe"], "STRIPE_SECRET_KEY")
	}
	if sm.config.Stripe.PublishableKey == "" {
		keys["stripe"] = append(keys["stripe"], "STRIPE_PUBLISHABLE_KEY")
	}
	if sm.config.Stripe.WebhookSecret == "" {
		keys["stripe"] = append(keys["stripe"], "STRIPE_WEBHOOK_SECRET")
	}
	
	if sm.config.USDC.RPCURL == "" {
		keys["usdc"] = append(keys["usdc"], "USDC_RPC_URL")
	}
	if sm.config.USDC.ContractAddr == "" {
		keys["usdc"] = append(keys["usdc"], "USDC_CONTRACT_ADDRESS")
	}
	if sm.config.USDC.PrivateKey == "" {
		keys["usdc"] = append(keys["usdc"], "USDC_PRIVATE_KEY")
	}
	
	// KYC services
	if sm.config.Jumio.APIKey == "" {
		keys["jumio"] = append(keys["jumio"], "JUMIO_API_KEY")
	}
	if sm.config.Jumio.APISecret == "" {
		keys["jumio"] = append(keys["jumio"], "JUMIO_API_SECRET")
	}
	
	// Support services
	if sm.config.Zendesk.Domain == "" {
		keys["zendesk"] = append(keys["zendesk"], "ZENDESK_DOMAIN")
	}
	if sm.config.Zendesk.Email == "" {
		keys["zendesk"] = append(keys["zendesk"], "ZENDESK_EMAIL")
	}
	if sm.config.Zendesk.APIToken == "" {
		keys["zendesk"] = append(keys["zendesk"], "ZENDESK_API_TOKEN")
	}
	
	return keys
}

// GetServiceInstructions returns instructions for setting up each service
func (sm *ServiceManager) GetServiceInstructions() map[string]string {
	instructions := make(map[string]string)
	
	instructions["stripe"] = `
1. Go to https://stripe.com and create an account
2. Go to Developers > API Keys
3. Copy the Secret key and Publishable key
4. Set environment variables:
   - STRIPE_SECRET_KEY=sk_test_...
   - STRIPE_PUBLISHABLE_KEY=pk_test_...
   - STRIPE_WEBHOOK_SECRET=whsec_...
	`
	
	instructions["usdc"] = `
1. Get an Ethereum RPC URL from Infura, Alchemy, or similar
2. Get the USDC contract address for your network
3. Generate a private key for the protocol wallet
4. Set environment variables:
   - USDC_RPC_URL=https://mainnet.infura.io/v3/YOUR_PROJECT_ID
   - USDC_CONTRACT_ADDRESS=0xA0b86a33E6441b8c4C8C0C4C0C4C0C4C0C4C0C4C
   - USDC_PRIVATE_KEY=0x...
	`
	
	instructions["jumio"] = `
1. Go to https://www.jumio.com and create an account
2. Go to API Credentials
3. Copy the API key and secret
4. Set environment variables:
   - JUMIO_API_KEY=your_api_key
   - JUMIO_API_SECRET=your_api_secret
	`
	
	instructions["zendesk"] = `
1. Go to https://www.zendesk.com and create an account
2. Go to Admin > API
3. Enable API access and generate a token
4. Set environment variables:
   - ZENDESK_DOMAIN=yourcompany.zendesk.com
   - ZENDESK_EMAIL=admin@yourcompany.com
   - ZENDESK_API_TOKEN=your_api_token
	`
	
	return instructions
}
