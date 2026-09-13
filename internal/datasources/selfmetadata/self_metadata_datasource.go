package selfmetadata

import (
	"context"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/maxvanp/terraform-provider-graviteeam/internal/client"
	"github.com/maxvanp/terraform-provider-graviteeam/internal/datasources/jsonformat"
)

var _ datasource.DataSource = &SelfMetadataDataSource{}

type SelfMetadataDataSource struct {
	client *client.Client
}

type SelfMetadataModel struct {
	Kind       types.String `tfsdk:"kind"`
	ResultJSON types.String `tfsdk:"result_json"`
}

var selfMetadataPaths = map[string]string{
	"current_user":  "",
	"notifications": "/notifications",
}

func NewSelfMetadataDataSource() datasource.DataSource {
	return &SelfMetadataDataSource{}
}

func (d *SelfMetadataDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_self_metadata"
}

func (d *SelfMetadataDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Reads Gravitee AM metadata for the authenticated user such as profile and notifications",
		Attributes: map[string]schema.Attribute{
			"kind": schema.StringAttribute{
				Required:    true,
				Description: "Self metadata kind. Supported values: " + selfMetadataKindList(),
			},
			"result_json": schema.StringAttribute{
				Computed:    true,
				Description: "JSON string containing the self metadata response",
			},
		},
	}
}

func (d *SelfMetadataDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *SelfMetadataDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config SelfMetadataModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	path, ok := selfMetadataPaths[config.Kind.ValueString()]
	if !ok {
		resp.Diagnostics.AddError("Unsupported self metadata kind", "Expected one of: "+selfMetadataKindList())
		return
	}

	raw, err := d.client.GetSelfMetadata(ctx, path)
	if err != nil {
		resp.Diagnostics.AddError("Error reading self metadata", err.Error())
		return
	}
	result := formatJSON(raw)

	config.ResultJSON = types.StringValue(result)
	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}

func formatJSON(raw []byte) string {
	return jsonformat.String(raw)
}

func selfMetadataKindList() string {
	kinds := make([]string, 0, len(selfMetadataPaths))
	for kind := range selfMetadataPaths {
		kinds = append(kinds, kind)
	}
	sort.Strings(kinds)
	return strings.Join(kinds, ", ")
}
