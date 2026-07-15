package analytics

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/maxvanp/terraform-provider-graviteeam/internal/client"
)

var _ datasource.DataSource = &AnalyticsDataSource{}

type AnalyticsDataSource struct {
	client *client.Client
}

type AnalyticsModel struct {
	DomainID types.String `tfsdk:"domain_id"`
	Type     types.String `tfsdk:"type"`
	Field    types.String `tfsdk:"field"`
	From     types.Int64  `tfsdk:"from"`
	To       types.Int64  `tfsdk:"to"`
	Interval types.Int64  `tfsdk:"interval"`
	Size     types.Int64  `tfsdk:"size"`
	Result   types.String `tfsdk:"result"`
}

func NewAnalyticsDataSource() datasource.DataSource {
	return &AnalyticsDataSource{}
}

func (d *AnalyticsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_analytics"
}

func (d *AnalyticsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Reads analytics data from a Gravitee AM domain",
		Attributes: map[string]schema.Attribute{
			"domain_id": schema.StringAttribute{
				Required:    true,
				Description: "The ID of the domain to read analytics from",
			},
			"type": schema.StringAttribute{
				Required:    true,
				Description: "Analytics type: DATE_HISTO, COUNT, or GROUP_BY",
			},
			"field": schema.StringAttribute{
				Optional:    true,
				Description: "Aggregation field",
			},
			"from": schema.Int64Attribute{
				Optional:    true,
				Description: "Start timestamp (milliseconds)",
			},
			"to": schema.Int64Attribute{
				Optional:    true,
				Description: "End timestamp (milliseconds)",
			},
			"interval": schema.Int64Attribute{
				Optional:    true,
				Description: "Histogram interval (milliseconds)",
			},
			"size": schema.Int64Attribute{
				Optional:    true,
				Description: "Number of results to return",
			},
			"result": schema.StringAttribute{
				Computed:    true,
				Description: "JSON string containing the analytics result (structure varies by type)",
			},
		},
	}
}

func (d *AnalyticsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *AnalyticsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config AnalyticsModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := buildAnalyticsParams(config, time.Now().UnixMilli())

	result, err := d.client.GetAnalytics(ctx, config.DomainID.ValueString(), params)
	if err != nil {
		// Gravitee AM returns 500 when there is no analytics data yet (e.g. freshly created domain).
		// Treat this as an empty result rather than failing.
		if strings.Contains(err.Error(), "status 500") {
			config.Result = types.StringValue("{}")
			resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
			return
		}
		resp.Diagnostics.AddError("Error reading analytics", err.Error())
		return
	}

	resultJSON := formatAnalyticsResult(result)

	config.Result = types.StringValue(resultJSON)
	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}

func buildAnalyticsParams(config AnalyticsModel, now int64) map[string]string {
	params := map[string]string{
		"type": config.Type.ValueString(),
	}
	if !config.Field.IsNull() && !config.Field.IsUnknown() {
		params["field"] = config.Field.ValueString()
	}
	if !config.From.IsNull() && !config.From.IsUnknown() {
		params["from"] = fmt.Sprintf("%d", config.From.ValueInt64())
	} else {
		params["from"] = fmt.Sprintf("%d", now-86400000)
	}
	if !config.To.IsNull() && !config.To.IsUnknown() {
		params["to"] = fmt.Sprintf("%d", config.To.ValueInt64())
	} else {
		params["to"] = fmt.Sprintf("%d", now)
	}
	if !config.Interval.IsNull() && !config.Interval.IsUnknown() {
		params["interval"] = fmt.Sprintf("%d", config.Interval.ValueInt64())
	}
	if !config.Size.IsNull() && !config.Size.IsUnknown() {
		params["size"] = fmt.Sprintf("%d", config.Size.ValueInt64())
	}
	return params
}

func formatAnalyticsResult(value interface{}) string {
	resultJSON, _ := json.MarshalIndent(value, "", "  ")
	return string(resultJSON)
}
