package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ provider.Provider = &OCXProvider{}

// OCXProvider defines the provider implementation.
type OCXProvider struct {
	version string
}

// OCXProviderModel describes the provider data model.
type OCXProviderModel struct {
	ServerURL types.String `tfsdk:"server_url"`
	APIKey    types.String `tfsdk:"api_key"`
}

func (p *OCXProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "ocx"
	resp.Version = p.version
}

func (p *OCXProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "OCX provider",
		Attributes: map[string]schema.Attribute{
			"server_url": schema.StringAttribute{
				MarkdownDescription: "OCX server URL",
				Optional:            true,
			},
			"api_key": schema.StringAttribute{
				MarkdownDescription: "OCX API key",
				Optional:            true,
				Sensitive:           true,
			},
		},
	}
}

func (p *OCXProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data OCXProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	serverURL := os.Getenv("OCX_SERVER_URL")
	if !data.ServerURL.IsNull() {
		serverURL = data.ServerURL.ValueString()
	}

	apiKey := os.Getenv("OCX_API_KEY")
	if !data.APIKey.IsNull() {
		apiKey = data.APIKey.ValueString()
	}

	if serverURL == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("server_url"),
			"Missing OCX Server URL",
			"Set server_url or OCX_SERVER_URL environment variable",
		)
	}

	if apiKey == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_key"),
			"Missing OCX API Key",
			"Set api_key or OCX_API_KEY environment variable",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	client := &OCXClient{
		ServerURL: serverURL,
		APIKey:    apiKey,
	}

	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *OCXProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewProvenanceResource,
	}
}

func (p *OCXProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		// Add data sources here
	}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &OCXProvider{
			version: version,
		}
	}
}

// OCXClient represents the client for OCX API
type OCXClient struct {
	ServerURL string
	APIKey    string
}

// NewProvenanceResource creates a new provenance resource
func NewProvenanceResource() resource.Resource {
	return &ProvenanceResource{}
}

// ProvenanceResource defines the resource implementation
type ProvenanceResource struct {
	client *OCXClient
}

func (r *ProvenanceResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_provenance"
}

func (r *ProvenanceResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "OCX provenance resource",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Provenance identifier",
				Computed:            true,
			},
			"trigger_hash": schema.StringAttribute{
				MarkdownDescription: "Trigger hash",
				Required:            true,
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
		resp.Diagnostics.AddError("Unexpected Resource Configure Type", "Expected *OCXClient")
		return
	}

	r.client = client
}

func (r *ProvenanceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Implementation here - simplified for compatibility
	resp.Diagnostics.AddError("Not Implemented", "Create operation not yet implemented")
}

func (r *ProvenanceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Implementation here - simplified for compatibility
}

func (r *ProvenanceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Update Not Supported", "Provenance records are immutable")
}

func (r *ProvenanceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Implementation here - simplified for compatibility
}
