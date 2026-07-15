package audits

import (
	"context"
	"encoding/json"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/maxvanp/terraform-provider-graviteeam/internal/client"
)

var _ datasource.DataSource = &AuditsDataSource{}

type AuditsDataSource struct {
	client *client.Client
}

type AuditsModel struct {
	DomainID types.String `tfsdk:"domain_id"`
	Size     types.Int64  `tfsdk:"size"`
	Audits   types.String `tfsdk:"audits"`
}

func NewAuditsDataSource() datasource.DataSource {
	return &AuditsDataSource{}
}

func (d *AuditsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_audits"
}

func (d *AuditsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Reads recent audit logs from a Gravitee AM domain",
		Attributes: map[string]schema.Attribute{
			"domain_id": schema.StringAttribute{
				Required:    true,
				Description: "The ID of the domain to read audits from",
			},
			"size": schema.Int64Attribute{
				Optional:    true,
				Description: "Number of audit entries to retrieve (default: 10)",
			},
			"audits": schema.StringAttribute{
				Computed:    true,
				Description: "JSON string containing the audit log entries",
			},
		},
	}
}

func (d *AuditsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *AuditsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config AuditsModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	size := auditSize(config)

	result, err := d.client.ListAudits(ctx, config.DomainID.ValueString(), 0, size)
	if err != nil {
		resp.Diagnostics.AddError("Error reading audits", err.Error())
		return
	}

	auditsJSON := formatAuditEntries(result)

	config.Audits = types.StringValue(auditsJSON)
	if config.Size.IsNull() || config.Size.IsUnknown() {
		config.Size = types.Int64Value(int64(size))
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}

func auditSize(config AuditsModel) int {
	if !config.Size.IsNull() && !config.Size.IsUnknown() {
		return int(config.Size.ValueInt64())
	}
	return 10
}

func formatAuditEntries(result map[string]interface{}) string {
	entries := extractAuditEntries(result)
	auditsJSON, _ := json.MarshalIndent(entries, "", "  ")
	return string(auditsJSON)
}

func extractAuditEntries(result map[string]interface{}) []interface{} {
	if data, ok := result["data"].([]interface{}); ok {
		return data
	}
	return nil
}
