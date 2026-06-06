package plugins

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/maxvanp/terraform-provider-graviteeam/internal/client"
)

var _ datasource.DataSource = &PluginsDataSource{}

type PluginsDataSource struct {
	client *client.Client
}

type PluginsModel struct {
	Category   types.String `tfsdk:"category"`
	PluginID   types.String `tfsdk:"plugin_id"`
	Schema     types.Bool   `tfsdk:"schema"`
	ResultJSON types.String `tfsdk:"result_json"`
}

var allowedPluginCategories = map[string]struct{}{
	"auth-device-notifiers": {},
	"authorization-engines": {},
	"bot-detections":        {},
	"certificates":          {},
	"device-identifiers":    {},
	"extensionGrants":       {},
	"factors":               {},
	"identities":            {},
	"notifiers":             {},
	"policies":              {},
	"reporters":             {},
	"resources":             {},
}

func NewPluginsDataSource() datasource.DataSource {
	return &PluginsDataSource{}
}

func (d *PluginsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_plugins"
}

func (d *PluginsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Reads Gravitee AM platform plugin catalogs, plugin details, or plugin configuration schemas",
		Attributes: map[string]schema.Attribute{
			"category": schema.StringAttribute{
				Required:    true,
				Description: "Plugin category, such as identities, factors, certificates, resources, reporters, policies, or notifiers",
			},
			"plugin_id": schema.StringAttribute{
				Optional:    true,
				Description: "Optional plugin ID. When omitted, the data source reads the plugin catalog for the category.",
			},
			"schema": schema.BoolAttribute{
				Optional:    true,
				Description: "Whether to read the selected plugin configuration schema. Requires plugin_id.",
			},
			"result_json": schema.StringAttribute{
				Computed:    true,
				Description: "JSON string containing the plugin catalog, plugin details, plugin schema, or null when the API returns no content",
			},
		},
	}
}

func (d *PluginsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *PluginsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config PluginsModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	category := config.Category.ValueString()
	if _, ok := allowedPluginCategories[category]; !ok {
		resp.Diagnostics.AddError("Unsupported plugin category", "Expected one of: "+pluginCategoryList())
		return
	}

	pluginID := ""
	if !config.PluginID.IsNull() && !config.PluginID.IsUnknown() {
		pluginID = config.PluginID.ValueString()
	}
	readSchema := !config.Schema.IsNull() && !config.Schema.IsUnknown() && config.Schema.ValueBool()
	if readSchema && pluginID == "" {
		resp.Diagnostics.AddError("Missing plugin_id", "plugin_id is required when schema is true")
		return
	}

	rawJSON, err := d.client.GetPlatformPlugin(ctx, category, pluginID, readSchema)
	if err != nil {
		resp.Diagnostics.AddError("Error reading platform plugin", err.Error())
		return
	}

	resultJSON, err := formatJSONResult(rawJSON)
	if err != nil {
		resp.Diagnostics.AddError("Error formatting platform plugin response", err.Error())
		return
	}

	config.ResultJSON = types.StringValue(resultJSON)
	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}

func formatJSONResult(raw []byte) (string, error) {
	if len(raw) == 0 {
		return "null", nil
	}
	var parsed interface{}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return "", err
	}
	formatted, err := json.MarshalIndent(parsed, "", "  ")
	if err != nil {
		return "", err
	}
	return string(formatted), nil
}

func pluginCategoryList() string {
	categories := make([]string, 0, len(allowedPluginCategories))
	for category := range allowedPluginCategories {
		categories = append(categories, category)
	}
	return strings.Join(categories, ", ")
}
