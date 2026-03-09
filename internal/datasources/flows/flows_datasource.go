package flows

import (
	"context"
	"encoding/json"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/maxvanp/terraform-provider-graviteeam/internal/client"
)

var _ datasource.DataSource = &FlowsDataSource{}

type FlowsDataSource struct {
	client *client.Client
}

type FlowsModel struct {
	DomainID types.String `tfsdk:"domain_id"`
	Flows    types.String `tfsdk:"flows"`
}

func NewFlowsDataSource() datasource.DataSource {
	return &FlowsDataSource{}
}

func (d *FlowsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_flows"
}

func (d *FlowsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Reads flows from a Gravitee AM domain",
		Attributes: map[string]schema.Attribute{
			"domain_id": schema.StringAttribute{
				Required:    true,
				Description: "The ID of the domain to read flows from",
			},
			"flows": schema.StringAttribute{
				Computed:    true,
				Description: "JSON string containing the flow definitions",
			},
		},
	}
}

func (d *FlowsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *FlowsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config FlowsModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := d.client.ListFlows(ctx, config.DomainID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading flows", err.Error())
		return
	}

	flowsJSON, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		resp.Diagnostics.AddError("Error marshaling flows", err.Error())
		return
	}

	config.Flows = types.StringValue(string(flowsJSON))
	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
