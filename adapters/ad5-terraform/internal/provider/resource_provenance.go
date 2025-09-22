package provider

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &ProvenanceResource{}

func NewProvenanceResource() resource.Resource {
	return &ProvenanceResource{}
}

type ProvenanceResource struct {
	client *OCXClient
}

type ProvenanceResourceModel struct {
	ID           types.String `tfsdk:"id"`
	TriggerHash  types.String `tfsdk:"trigger_hash"`
	OCXServer    types.String `tfsdk:"ocx_server"`
	Workspace    types.String `tfsdk:"workspace"`
	RunID        types.String `tfsdk:"run_id"`
	ReceiptHash  types.String `tfsdk:"receipt_hash"`
	ReceiptData  types.String `tfsdk:"receipt_data"`
	StorageURL   types.String `tfsdk:"storage_url"`
	CreatedAt    types.String `tfsdk:"created_at"`
}

func (r *ProvenanceResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_provenance"
}

func (r *ProvenanceResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "OCX provenance resource for Terraform apply operations",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Unique identifier for the provenance record",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"trigger_hash": schema.StringAttribute{
				MarkdownDescription: "Hash that triggers provenance generation (e.g., plan hash)",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"ocx_server": schema.StringAttribute{
				MarkdownDescription: "OCX server URL (overrides provider config)",
				Optional:            true,
			},
			"workspace": schema.StringAttribute{
				MarkdownDescription: "Terraform workspace name",
				Optional:            true,
				Computed:            true,
			},
			"run_id": schema.StringAttribute{
				MarkdownDescription: "Terraform run ID",
				Optional:            true,
				Computed:            true,
			},
			"receipt_hash": schema.StringAttribute{
				MarkdownDescription: "SHA-256 hash of the generated receipt",
				Computed:            true,
			},
			"receipt_data": schema.StringAttribute{
				MarkdownDescription: "Base64-encoded OCX receipt",
				Computed:            true,
			},
			"storage_url": schema.StringAttribute{
				MarkdownDescription: "URL where the receipt is stored",
				Optional:            true,
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "Timestamp when provenance was created",
				Computed:            true,
			},
		},
	}
}

func (r *ProvenanceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*OCXClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *OCXClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}

func (r *ProvenanceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ProvenanceResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Generate unique ID
	provenance_id := fmt.Sprintf("ocx-provenance-%d", time.Now().Unix())
	
	// Get workspace and run info from Terraform environment
	workspace := "default"
	if w := data.Workspace.ValueString(); w != "" {
		workspace = w
	}
	
	runID := fmt.Sprintf("tf-run-%d", time.Now().Unix())
	if r := data.RunID.ValueString(); r != "" {
		runID = r
	}

	// Create provenance context
	context := map[string]interface{}{
		"trigger_hash": data.TriggerHash.ValueString(),
		"workspace":    workspace,
		"run_id":       runID,
		"provider":     "terraform",
		"timestamp":    time.Now().UTC().Format(time.RFC3339),
	}

	contextBytes, _ := json.Marshal(context)
	
	// Create verification request
	verificationReq := map[string]interface{}{
		"artifact": data.TriggerHash.ValueString(),
		"input":    string(contextBytes),
		"cycles":   50000,
	}

	// Send to OCX server
	receipt, err := r.client.ExecuteVerification(ctx, verificationReq)
	if err != nil {
		resp.Diagnostics.AddError("OCX Verification Failed", err.Error())
		return
	}

	// Calculate receipt hash
	receiptBytes, _ := json.Marshal(receipt)
	receiptHash := fmt.Sprintf("%x", sha256.Sum256(receiptBytes))

	// Store receipt if storage URL provided
	storageURL := ""
	if data.StorageURL.ValueString() != "" {
		storageURL, err = r.storeReceipt(ctx, data.StorageURL.ValueString(), receiptBytes, provenance_id)
		if err != nil {
			resp.Diagnostics.AddWarning("Receipt Storage Failed", err.Error())
		}
	}

	// Set computed values
	data.ID = types.StringValue(provenance_id)
	data.Workspace = types.StringValue(workspace)
	data.RunID = types.StringValue(runID)
	data.ReceiptHash = types.StringValue(receiptHash)
	data.ReceiptData = types.StringValue(string(receiptBytes))
	data.StorageURL = types.StringValue(storageURL)
	data.CreatedAt = types.StringValue(time.Now().UTC().Format(time.RFC3339))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ProvenanceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ProvenanceResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// OCX receipts are immutable, so we just verify the state is consistent
	if data.ReceiptData.ValueString() != "" {
		receiptBytes := []byte(data.ReceiptData.ValueString())
		receiptHash := fmt.Sprintf("%x", sha256.Sum256(receiptBytes))
		
		if receiptHash != data.ReceiptHash.ValueString() {
			resp.Diagnostics.AddError(
				"Receipt Hash Mismatch",
				"The stored receipt hash does not match the calculated hash. The receipt may have been tampered with.",
			)
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ProvenanceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError(
		"Update Not Supported",
		"OCX provenance records are immutable and cannot be updated. Any changes require replacement.",
	)
}

func (r *ProvenanceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// OCX receipts are permanent records - we don't actually delete them
	// We just remove them from Terraform state
	// The receipt remains in the OCX audit trail
}

func (r *ProvenanceResource) storeReceipt(ctx context.Context, storageURL string, receipt []byte, provenanceID string) (string, error) {
	// Implementation would depend on storage backend (S3, GCS, etc.)
	// For now, return the input URL as a placeholder
	return fmt.Sprintf("%s/%s.json", storageURL, provenanceID), nil
}
