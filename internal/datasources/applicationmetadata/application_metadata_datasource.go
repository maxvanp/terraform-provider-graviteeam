package applicationmetadata

import (
	"context"
	"encoding/json"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/maxvanp/terraform-provider-graviteeam/internal/client"
)

var _ datasource.DataSource = &ApplicationMetadataDataSource{}

type ApplicationMetadataDataSource struct {
	client *client.Client
}

type ApplicationMetadataModel struct {
	DomainID      types.String `tfsdk:"domain_id"`
	ApplicationID types.String `tfsdk:"application_id"`
	Kind          types.String `tfsdk:"kind"`
	Type          types.String `tfsdk:"type"`
	Field         types.String `tfsdk:"field"`
	From          types.Int64  `tfsdk:"from"`
	To            types.Int64  `tfsdk:"to"`
	Interval      types.Int64  `tfsdk:"interval"`
	Size          types.Int64  `tfsdk:"size"`
	ResultJSON    types.String `tfsdk:"result_json"`
}

var applicationMetadataPaths = map[string]func(ApplicationMetadataModel) (string, error){
	"analytics": func(config ApplicationMetadataModel) (string, error) {
		return "/analytics" + analyticsQuery(config), nil
	},
	"resources": func(config ApplicationMetadataModel) (string, error) {
		return "/resources" + pagingQuery(config), nil
	},
}

func NewApplicationMetadataDataSource() datasource.DataSource {
	return &ApplicationMetadataDataSource{}
}

func (d *ApplicationMetadataDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_application_metadata"
}

func (d *ApplicationMetadataDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Reads Gravitee AM application metadata such as application analytics and UMA resources",
		Attributes: map[string]schema.Attribute{
			"domain_id": schema.StringAttribute{
				Required:    true,
				Description: "The ID of the domain",
			},
			"application_id": schema.StringAttribute{
				Required:    true,
				Description: "The ID of the application",
			},
			"kind": schema.StringAttribute{
				Required:    true,
				Description: "Application metadata kind. Supported values: " + applicationMetadataKindList(),
			},
			"type": schema.StringAttribute{
				Optional:    true,
				Description: "Analytics type for analytics kind (default: GROUP_BY)",
			},
			"field": schema.StringAttribute{
				Optional:    true,
				Description: "Analytics aggregation field for analytics kind",
			},
			"from": schema.Int64Attribute{
				Optional:    true,
				Description: "Analytics start timestamp in milliseconds",
			},
			"to": schema.Int64Attribute{
				Optional:    true,
				Description: "Analytics end timestamp in milliseconds",
			},
			"interval": schema.Int64Attribute{
				Optional:    true,
				Description: "Analytics histogram interval in milliseconds",
			},
			"size": schema.Int64Attribute{
				Optional:    true,
				Description: "Maximum entries to retrieve for paged or grouped responses",
			},
			"result_json": schema.StringAttribute{
				Computed:    true,
				Description: "JSON string containing the application metadata response",
			},
		},
	}
}

func (d *ApplicationMetadataDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ApplicationMetadataDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config ApplicationMetadataModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	pathBuilder, ok := applicationMetadataPaths[config.Kind.ValueString()]
	if !ok {
		resp.Diagnostics.AddError("Unsupported application metadata kind", "Expected one of: "+applicationMetadataKindList())
		return
	}
	path, err := pathBuilder(config)
	if err != nil {
		resp.Diagnostics.AddError("Invalid application metadata configuration", err.Error())
		return
	}

	raw, err := d.client.GetApplicationMetadata(ctx, config.DomainID.ValueString(), config.ApplicationID.ValueString(), path)
	if err != nil {
		if config.Kind.ValueString() == "analytics" && emptyAnalyticsError(err) {
			config.ResultJSON = types.StringValue("{}")
			resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
			return
		}
		resp.Diagnostics.AddError("Error reading application metadata", err.Error())
		return
	}
	result, err := formatJSON(raw)
	if err != nil {
		resp.Diagnostics.AddError("Error formatting application metadata", err.Error())
		return
	}

	config.ResultJSON = types.StringValue(result)
	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}

func emptyAnalyticsError(err error) bool {
	message := err.Error()
	return strings.Contains(message, "status 500") ||
		(strings.Contains(message, "status 400") && strings.Contains(message, "Malformed json"))
}

func analyticsQuery(config ApplicationMetadataModel) string {
	query := url.Values{}
	analyticsType := "GROUP_BY"
	if !config.Type.IsNull() && !config.Type.IsUnknown() && config.Type.ValueString() != "" {
		analyticsType = config.Type.ValueString()
	}
	query.Set("type", analyticsType)

	if !config.Field.IsNull() && !config.Field.IsUnknown() && config.Field.ValueString() != "" {
		query.Set("field", config.Field.ValueString())
	} else if analyticsType == "GROUP_BY" {
		query.Set("field", "application")
	}

	now := time.Now().UnixMilli()
	from := now - 86400000
	if !config.From.IsNull() && !config.From.IsUnknown() {
		from = config.From.ValueInt64()
	}
	to := now
	if !config.To.IsNull() && !config.To.IsUnknown() {
		to = config.To.ValueInt64()
	}
	query.Set("from", strconv.FormatInt(from, 10))
	query.Set("to", strconv.FormatInt(to, 10))

	if !config.Interval.IsNull() && !config.Interval.IsUnknown() {
		query.Set("interval", strconv.FormatInt(config.Interval.ValueInt64(), 10))
	}
	if !config.Size.IsNull() && !config.Size.IsUnknown() {
		query.Set("size", strconv.FormatInt(config.Size.ValueInt64(), 10))
	}

	return "?" + query.Encode()
}

func pagingQuery(config ApplicationMetadataModel) string {
	query := url.Values{}
	query.Set("page", "0")
	size := int64(50)
	if !config.Size.IsNull() && !config.Size.IsUnknown() {
		size = config.Size.ValueInt64()
	}
	query.Set("size", strconv.FormatInt(size, 10))
	return "?" + query.Encode()
}

func formatJSON(raw []byte) (string, error) {
	if len(raw) == 0 {
		return "null", nil
	}
	var parsed interface{}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		formatted, marshalErr := json.Marshal(string(raw))
		if marshalErr != nil {
			return "", marshalErr
		}
		return string(formatted), nil
	}
	formatted, err := json.MarshalIndent(parsed, "", "  ")
	if err != nil {
		return "", err
	}
	return string(formatted), nil
}

func applicationMetadataKindList() string {
	kinds := make([]string, 0, len(applicationMetadataPaths))
	for kind := range applicationMetadataPaths {
		kinds = append(kinds, kind)
	}
	sort.Strings(kinds)
	return strings.Join(kinds, ", ")
}
