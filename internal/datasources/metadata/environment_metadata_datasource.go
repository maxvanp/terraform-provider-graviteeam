package metadata

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/maxvanp/terraform-provider-graviteeam/internal/client"
)

var _ datasource.DataSource = &EnvironmentMetadataDataSource{}

type EnvironmentMetadataDataSource struct {
	client *client.Client
}

type EnvironmentMetadataModel struct {
	Kind       types.String `tfsdk:"kind"`
	ResultJSON types.String `tfsdk:"result_json"`
}

var environmentMetadataPaths = map[string]string{
	"data_planes":  "data-planes",
	"data_sources": "data-sources",
}

func NewEnvironmentMetadataDataSource() datasource.DataSource {
	return &EnvironmentMetadataDataSource{}
}

func (d *EnvironmentMetadataDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_environment_metadata"
}

func (d *EnvironmentMetadataDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Reads Gravitee AM environment metadata such as data planes and data sources",
		Attributes: map[string]schema.Attribute{
			"kind": schema.StringAttribute{
				Required:    true,
				Description: "Metadata kind. Supported values: " + metadataKindList(environmentMetadataPaths),
			},
			"result_json": schema.StringAttribute{
				Computed:    true,
				Description: "JSON string containing the environment metadata response",
			},
		},
	}
}

func (d *EnvironmentMetadataDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected DataSource Configure Type", "Expected *client.Client")
		return
	}
	d.client = c
}

func (d *EnvironmentMetadataDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config EnvironmentMetadataModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	path, ok := environmentMetadataPaths[config.Kind.ValueString()]
	if !ok {
		resp.Diagnostics.AddError("Unsupported environment metadata kind", "Expected one of: "+metadataKindList(environmentMetadataPaths))
		return
	}

	raw, err := d.client.GetEnvironmentMetadata(ctx, path)
	if err != nil {
		resp.Diagnostics.AddError("Error reading environment metadata", err.Error())
		return
	}
	result, err := formatJSON(raw)
	if err != nil {
		resp.Diagnostics.AddError("Error formatting environment metadata", err.Error())
		return
	}

	config.ResultJSON = types.StringValue(result)
	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
