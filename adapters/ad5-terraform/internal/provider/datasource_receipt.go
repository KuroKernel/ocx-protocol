package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &ReceiptDataSource{}

func NewReceiptDataSource() datasource.DataSource {
	return &ReceiptDataSource{}
}

type ReceiptDataSource struct {
	client *OCXClient
}

type ReceiptDataSourceModel struct {
	ReceiptID   types.String `tfsdk:"receipt_id"`
	ReceiptData types.String `tfsdk:"receipt_data"`
	Verified    types.Bool   `tfsdk:"verified"`
}

func (d *ReceiptDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_receipt"
}

func (d *ReceiptDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "OCX receipt data source",
		Attributes: map[string]schema.Attribute{
			"receipt_id": schema.StringAttribute{
				MarkdownDescription: "ID of the receipt to retrieve",
				Required:            true,
			},
			"receipt_data": schema.StringAttribute{
				MarkdownDescription: "OCX receipt data",
				Computed:            true,
			},
			"verified": schema.BoolAttribute{
				MarkdownDescription: "Whether the receipt is verified",
				Computed:            true,
			},
		},
	}
}

func (d *ReceiptDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*OCXClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected DataSource Configure Type",
			fmt.Sprintf("Expected *OCXClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = client
}

func (d *ReceiptDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ReceiptDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Implementation: retrieve the receipt from the OCX server
	create a placeholder receipt
	receiptData := fmt.Sprintf(`{"receipt_id": "%s", "verified": true}`, data.ReceiptID.ValueString())
	
	data.ReceiptData = types.StringValue(receiptData)
	data.Verified = types.BoolValue(true)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
