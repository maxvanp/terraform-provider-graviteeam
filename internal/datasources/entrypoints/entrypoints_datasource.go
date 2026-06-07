package entrypoints

import (
	"context"
	"encoding/json"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/maxvanp/terraform-provider-graviteeam/internal/client"
)

var _ datasource.DataSource = &EntrypointsDataSource{}

type EntrypointsDataSource struct {
	client *client.Client
}

type EntrypointsModel struct {
	DomainID    types.String `tfsdk:"domain_id"`
	Entrypoints types.String `tfsdk:"entrypoints"`
}

func NewEntrypointsDataSource() datasource.DataSource {
	return &EntrypointsDataSource{}
}

func (d *EntrypointsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_entrypoints"
}

func (d *EntrypointsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Reads entrypoints from a Gravitee AM domain",
		Attributes: map[string]schema.Attribute{
			"domain_id": schema.StringAttribute{
				Required:    true,
				Description: "The ID of the domain to read entrypoints from",
			},
			"entrypoints": schema.StringAttribute{
				Computed:    true,
				Description: "JSON string containing the entrypoint definitions",
			},
		},
	}
}

func (d *EntrypointsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *EntrypointsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config EntrypointsModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	rawJSON, err := d.client.ListEntrypoints(ctx, config.DomainID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading entrypoints", err.Error())
		return
	}

	formatted, err := formatEntrypoints(rawJSON)
	if err != nil {
		resp.Diagnostics.AddError("Error formatting entrypoints", err.Error())
		return
	}

	config.Entrypoints = types.StringValue(formatted)
	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}

func formatEntrypoints(rawJSON []byte) (string, error) {
	var parsed interface{}
	if err := json.Unmarshal(rawJSON, &parsed); err != nil {
		return "", err
	}
	formatted, err := json.MarshalIndent(parsed, "", "  ")
	if err != nil {
		return "", err
	}
	return string(formatted), nil
}
