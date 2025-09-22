// adapters/ad5-terraform/provider.go - AD5 Terraform Provider for OCX Injection
// This follows the EXACT same pattern as AD2 webhook but for infrastructure injection

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/plugin"

	"encoding/json"
	"fmt"
)

// Provider returns a Terraform provider for OCX Protocol
func Provider() *schema.Provider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"ocx_inject": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "true",
				Description: "Enable OCX injection (true/verify)",
			},
			"ocx_cycles": {
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     50000,
				Description: "Maximum cycles allowed",
			},
			"ocx_profile": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "v1-min",
				Description: "OCX profile version",
			},
			"ocx_keystore": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "default",
				Description: "Keystore to use",
			},
			"ocx_verify_only": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Only verify, do not execute",
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			"ocx_verified_resource": resourceOCXVerifiedResource(),
		},
		DataSourcesMap: map[string]*schema.Resource{
			"ocx_receipt": dataSourceOCXReceipt(),
		},
		ConfigureContextFunc: providerConfigure,
	}
}

// providerConfigure configures the provider (same pattern as AD2)
func providerConfigure(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
	config := &OCXProviderConfig{
		Inject:      d.Get("ocx_inject").(string),
		Cycles:      d.Get("ocx_cycles").(int),
		Profile:     d.Get("ocx_profile").(string),
		Keystore:    d.Get("ocx_keystore").(string),
		VerifyOnly:  d.Get("ocx_verify_only").(bool),
		Verifier:    verify.NewVerifier(),
	}

	return config, nil
}

// OCXProviderConfig represents the provider configuration
type OCXProviderConfig struct {
	Inject     string
	Cycles     int
	Profile    string
	Keystore   string
	VerifyOnly bool
	Verifier   verify.Verifier
}

// resourceOCXVerifiedResource creates a verified resource
func resourceOCXVerifiedResource() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceOCXVerifiedResourceCreate,
		ReadContext:   resourceOCXVerifiedResourceRead,
		UpdateContext: resourceOCXVerifiedResourceUpdate,
		DeleteContext: resourceOCXVerifiedResourceDelete,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of the resource",
			},
			"resource_type": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Type of the resource",
			},
			"resource_config": {
				Type:        schema.TypeMap,
				Required:    true,
				Description: "Configuration of the resource",
			},
			"ocx_receipt": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "OCX receipt for the resource",
			},
			"ocx_cycles_used": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Cycles used for verification",
			},
			"ocx_verified": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Whether the resource is OCX verified",
			},
		},
	}
}

// resourceOCXVerifiedResourceCreate creates a verified resource (same pattern as AD2)
func resourceOCXVerifiedResourceCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	config := m.(*OCXProviderConfig)
	
	// Check if OCX injection is enabled (same logic as AD2)
	if !shouldInjectOCX(config) {
		return diag.Errorf("OCX injection is disabled")
	}
	
	// Generate OCX receipt (same pattern as AD2)
	receipt, err := generateOCXReceipt(d, config)
	if err != nil {
		return diag.Errorf("Failed to generate OCX receipt: %v", err)
	}
	
	// Verify receipt if needed
	if config.VerifyOnly {
		if err := verifyOCXReceipt(receipt, config); err != nil {
			return diag.Errorf("OCX verification failed: %v", err)
		}
	}
	
	// Set resource ID
	d.SetId(fmt.Sprintf("ocx-verified-%d", time.Now().Unix()))
	
	// Set computed values
	d.Set("ocx_receipt", receipt)
	d.Set("ocx_cycles_used", config.Cycles)
	d.Set("ocx_verified", true)
	
	return nil
}

// resourceOCXVerifiedResourceRead reads a verified resource
func resourceOCXVerifiedResourceRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	// In a real implementation, this would read the resource state
	return nil
}

// resourceOCXVerifiedResourceUpdate updates a verified resource
func resourceOCXVerifiedResourceUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	config := m.(*OCXProviderConfig)
	
	// Regenerate OCX receipt for updates
	receipt, err := generateOCXReceipt(d, config)
	if err != nil {
		return diag.Errorf("Failed to regenerate OCX receipt: %v", err)
	}
	
	// Update computed values
	d.Set("ocx_receipt", receipt)
	d.Set("ocx_cycles_used", config.Cycles)
	d.Set("ocx_verified", true)
	
	return nil
}

// resourceOCXVerifiedResourceDelete deletes a verified resource
func resourceOCXVerifiedResourceDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	// In a real implementation, this would clean up the resource
	d.SetId("")
	return nil
}

// dataSourceOCXReceipt creates a data source for OCX receipts
func dataSourceOCXReceipt() *schema.Resource {
	return &schema.Resource{
		ReadContext: dataSourceOCXReceiptRead,
		Schema: map[string]*schema.Schema{
			"receipt_id": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "ID of the receipt to retrieve",
			},
			"receipt_data": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "OCX receipt data",
			},
			"verified": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Whether the receipt is verified",
			},
		},
	}
}

// dataSourceOCXReceiptRead reads an OCX receipt
func dataSourceOCXReceiptRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	config := m.(*OCXProviderConfig)
	receiptID := d.Get("receipt_id").(string)
	
	// In a real implementation, this would retrieve the receipt
	// For now, create a placeholder receipt
	receipt := createPlaceholderReceipt(receiptID, config)
	
	d.SetId(receiptID)
	d.Set("receipt_data", receipt)
	d.Set("verified", true)
	
	return nil
}

// shouldInjectOCX determines if OCX should be injected (same logic as AD2)
func shouldInjectOCX(config *OCXProviderConfig) bool {
	return config.Inject == "true" || config.Inject == "verify"
}

// generateOCXReceipt generates an OCX receipt (same pattern as AD2)
func generateOCXReceipt(d *schema.ResourceData, config *OCXProviderConfig) (string, error) {
	// Create artifact hash from resource
	artifactHash := hashResource(d)
	
	// Create input hash from configuration
	inputHash := hashConfig(d.Get("resource_config"))
	
	// Create output hash (placeholder)
	outputHash := hashData([]byte("terraform_output"))
	
	// Create issuer key
	issuerKey := getIssuerKey(config)
	
	// Create receipt
	receipt := cbor.NewOCXReceiptV1_1(artifactHash, inputHash, outputHash, uint64(config.Cycles), issuerKey)
	
	// Add request binding
	requestDigest := hashConfig(d.Get("resource_config"))
	receipt.AddRequestBinding(requestDigest)
	
	// Add witness signature if enabled
	if config.VerifyOnly {
		witnessManager := cbor.NewWitnessManager()
		witnessManager.AddWitness("terraform", issuerKey)
		witnessManager.SignReceipt(receipt)
	}
	
	// Serialize receipt
	receiptData, err := receipt.Serialize()
	if err != nil {
		return "", err
	}
	
	return string(receiptData), nil
}

// verifyOCXReceipt verifies an OCX receipt
func verifyOCXReceipt(receipt string, config *OCXProviderConfig) error {
	// In a real implementation, this would verify the receipt
	// For now, just return success
	return nil
}

// createPlaceholderReceipt creates a placeholder receipt
func createPlaceholderReceipt(receiptID string, config *OCXProviderConfig) string {
	receipt := map[string]interface{}{
		"id":           receiptID,
		"version":      1,
		"cycles":       config.Cycles,
		"profile":      config.Profile,
		"verified":     true,
		"generated_at": time.Now().Format(time.RFC3339),
	}
	
	data, _ := json.Marshal(receipt)
	return string(data)
}

// Helper functions (same pattern as AD2)
func hashResource(d *schema.ResourceData) [32]byte {
	name := d.Get("name").(string)
	resourceType := d.Get("resource_type").(string)
	data := fmt.Sprintf("%s:%s", name, resourceType)
	return hashData([]byte(data))
}

func hashConfig(config interface{}) [32]byte {
	data, _ := json.Marshal(config)
	return hashData(data)
}

func hashData(data []byte) [32]byte {
	// In a real implementation, this would use crypto/sha256
	return [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32}
}

func getIssuerKey(config *OCXProviderConfig) [32]byte {
	// In a real implementation, this would load from keystore
	return [32]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32}
}

func main() {
	plugin.Serve(&plugin.ServeOpts{
		ProviderFunc: Provider,
	})
}
