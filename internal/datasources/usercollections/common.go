package usercollections

import (
	"context"
	"encoding/json"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/maxvanp/terraform-provider-graviteeam/internal/client"
)

type collectionDataSource struct {
	client      *client.Client
	typeSuffix  string
	collection  string
	resultName  string
	description string
}

type collectionModel struct {
	DomainID types.String `tfsdk:"domain_id"`
	UserID   types.String `tfsdk:"user_id"`
	Items    types.String `tfsdk:"items_json"`
}

func (d *collectionDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_" + d.typeSuffix
}

func (d *collectionDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: d.description,
		Attributes: map[string]schema.Attribute{
			"domain_id": schema.StringAttribute{
				Required:    true,
				Description: "The ID of the domain",
			},
			"user_id": schema.StringAttribute{
				Required:    true,
				Description: "The ID of the user",
			},
			"items_json": schema.StringAttribute{
				Computed:    true,
				Description: "JSON string containing the user " + d.resultName,
			},
		},
	}
}

func (d *collectionDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *collectionDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config collectionModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	rawJSON, err := d.client.ListUserCollection(ctx, config.DomainID.ValueString(), config.UserID.ValueString(), d.collection)
	if err != nil {
		resp.Diagnostics.AddError("Error reading user "+d.resultName, err.Error())
		return
	}

	var parsed interface{}
	if err := json.Unmarshal(rawJSON, &parsed); err != nil {
		resp.Diagnostics.AddError("Error parsing user "+d.resultName, err.Error())
		return
	}
	formatted, err := json.MarshalIndent(parsed, "", "  ")
	if err != nil {
		resp.Diagnostics.AddError("Error formatting user "+d.resultName, err.Error())
		return
	}

	config.Items = types.StringValue(string(formatted))
	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
