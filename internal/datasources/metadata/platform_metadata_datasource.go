package metadata

import (
	"context"
	"net/url"
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
	RoleID     types.String `tfsdk:"role_id"`
	ResultJSON types.String `tfsdk:"result_json"`
}

var platformMetadataPaths = map[string]func(PlatformMetadataModel) (string, error){
	"alert_service_status": func(_ PlatformMetadataModel) (string, error) { return "platform/configuration/alerts/status", nil },
	"audit_event_types":    func(_ PlatformMetadataModel) (string, error) { return "platform/audits/events", nil },
	"email_required": func(_ PlatformMetadataModel) (string, error) {
		return "platform/configuration/users/email-required", nil
	},
	"flow_schema":  func(_ PlatformMetadataModel) (string, error) { return "platform/configuration/flow/schema", nil },
	"installation": func(_ PlatformMetadataModel) (string, error) { return "platform/installation", nil },
	"license":      func(_ PlatformMetadataModel) (string, error) { return "platform/license", nil },
	"role": func(config PlatformMetadataModel) (string, error) {
		if config.RoleID.IsNull() || config.RoleID.IsUnknown() || config.RoleID.ValueString() == "" {
			return "", metadataError("role_id is required for role kind")
		}
		return "platform/roles/" + url.PathEscape(config.RoleID.ValueString()), nil
	},
	"spel_grammar": func(_ PlatformMetadataModel) (string, error) { return "platform/configuration/spel/grammar", nil },
}

type metadataError string

func (e metadataError) Error() string {
	return string(e)
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
				Description: "Metadata kind. Supported values: " + platformMetadataKindList(),
			},
			"role_id": schema.StringAttribute{
				Optional:    true,
				Description: "Platform role ID, required for role kind",
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

	pathBuilder, ok := platformMetadataPaths[config.Kind.ValueString()]
	if !ok {
		resp.Diagnostics.AddError("Unsupported platform metadata kind", "Expected one of: "+platformMetadataKindList())
		return
	}
	path, err := pathBuilder(config)
	if err != nil {
		resp.Diagnostics.AddError("Invalid platform metadata configuration", err.Error())
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

func platformMetadataKindList() string {
	kinds := make([]string, 0, len(platformMetadataPaths))
	for kind := range platformMetadataPaths {
		kinds = append(kinds, kind)
	}
	sort.Strings(kinds)
	return strings.Join(kinds, ", ")
}

func metadataKindList(paths map[string]string) string {
	kinds := make([]string, 0, len(paths))
	for kind := range paths {
		kinds = append(kinds, kind)
	}
	sort.Strings(kinds)
	return strings.Join(kinds, ", ")
}
