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

var _ provider.Provider = &OCXProvider{}

type OCXProvider struct {
	version string
}

type OCXProviderModel struct {
	ServerURL types.String `tfsdk:"server_url"`
	APIKey    types.String `tfsdk:"api_key"`
	Timeout   types.Int64  `tfsdk:"timeout"`
}

func (p *OCXProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "ocx"
	resp.Version = p.version
}

func (p *OCXProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
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
			"timeout": schema.Int64Attribute{
				MarkdownDescription: "Request timeout in seconds",
				Optional:            true,
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

	// Configuration values
	serverURL := os.Getenv("OCX_SERVER_URL")
	if !data.ServerURL.IsNull() {
		serverURL = data.ServerURL.ValueString()
	}

	apiKey := os.Getenv("OCX_API_KEY")
	if !data.APIKey.IsNull() {
		apiKey = data.APIKey.ValueString()
	}

	timeout := int64(5)
	if !data.Timeout.IsNull() {
		timeout = data.Timeout.ValueInt64()
	}

	if serverURL == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("server_url"),
			"Missing OCX Server URL",
			"The provider cannot create the OCX API client as there is a missing or empty value for the OCX server URL. "+
				"Set the server_url value in the configuration or use the OCX_SERVER_URL environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}

	if apiKey == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_key"),
			"Missing OCX API Key",
			"The provider cannot create the OCX API client as there is a missing or empty value for the OCX API key. "+
				"Set the api_key value in the configuration or use the OCX_API_KEY environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	client := &OCXClient{
		ServerURL: serverURL,
		APIKey:    apiKey,
		Timeout:   timeout,
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
		NewReceiptDataSource,
	}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &OCXProvider{
			version: version,
		}
	}
}
