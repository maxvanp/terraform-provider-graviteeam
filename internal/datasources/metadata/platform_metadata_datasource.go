package metadata

import (
	"context"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/maxvanp/terraform-provider-graviteeam/internal/client"
)

var _ datasource.DataSource = &PlatformMetadataDataSource{}

type PlatformMetadataDataSource struct {
	client *client.Client
}

type PlatformMetadataModel struct {
	Kind       types.String `tfsdk:"kind"`
	ResultJSON types.String `tfsdk:"result_json"`
}

var platformMetadataPaths = map[string]string{
	"alert_service_status": "platform/configuration/alerts/status",
	"audit_event_types":    "platform/audits/events",
	"email_required":       "platform/configuration/users/email-required",
	"flow_schema":          "platform/configuration/flow/schema",
	"installation":         "platform/installation",
	"license":              "platform/license",
	"spel_grammar":         "platform/configuration/spel/grammar",
}

func NewPlatformMetadataDataSource() datasource.DataSource {
	return &PlatformMetadataDataSource{}
}

func (d *PlatformMetadataDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_platform_metadata"
}

func (d *PlatformMetadataDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Reads Gravitee AM platform metadata such as installation, license, audit event types, and configuration schemas",
		Attributes: map[string]schema.Attribute{
			"kind": schema.StringAttribute{
				Required:    true,
				Description: "Metadata kind. Supported values: " + metadataKindList(platformMetadataPaths),
			},
			"result_json": schema.StringAttribute{
				Computed:    true,
				Description: "JSON string containing the platform metadata response",
			},
		},
	}
}

func (d *PlatformMetadataDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *PlatformMetadataDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config PlatformMetadataModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	path, ok := platformMetadataPaths[config.Kind.ValueString()]
	if !ok {
		resp.Diagnostics.AddError("Unsupported platform metadata kind", "Expected one of: "+metadataKindList(platformMetadataPaths))
		return
	}

	raw, err := d.client.GetPlatformMetadata(ctx, path)
	if err != nil {
		resp.Diagnostics.AddError("Error reading platform metadata", err.Error())
		return
	}
	result, err := formatJSON(raw)
	if err != nil {
		resp.Diagnostics.AddError("Error formatting platform metadata", err.Error())
		return
	}

	config.ResultJSON = types.StringValue(result)
	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}

func metadataKindList(paths map[string]string) string {
	kinds := make([]string, 0, len(paths))
	for kind := range paths {
		kinds = append(kinds, kind)
	}
	sort.Strings(kinds)
	return strings.Join(kinds, ", ")
}
