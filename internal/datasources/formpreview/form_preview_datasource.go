package formpreview

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/maxvanp/terraform-provider-graviteeam/internal/client"
)

var _ datasource.DataSource = &FormPreviewDataSource{}

type FormPreviewDataSource struct {
	client *client.Client
}

type FormPreviewModel struct {
	DomainID   types.String `tfsdk:"domain_id"`
	Template   types.String `tfsdk:"template"`
	Type       types.String `tfsdk:"type"`
	Content    types.String `tfsdk:"content"`
	ResultJSON types.String `tfsdk:"result_json"`
}

func NewFormPreviewDataSource() datasource.DataSource {
	return &FormPreviewDataSource{}
}

func (d *FormPreviewDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_form_preview"
}

func (d *FormPreviewDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Renders a Gravitee AM form or email template preview for a domain.",
		Attributes: map[string]schema.Attribute{
			"domain_id": schema.StringAttribute{
				Required:    true,
				Description: "The ID of the domain.",
			},
			"template": schema.StringAttribute{
				Required:    true,
				Description: "Template name to preview, for example LOGIN. The API preview endpoint expects lower-case template names; this data source normalizes the value before calling Gravitee AM.",
			},
			"type": schema.StringAttribute{
				Optional:    true,
				Description: "Preview type, either FORM or EMAIL. Defaults to FORM.",
			},
			"content": schema.StringAttribute{
				Required:    true,
				Description: "Template content to render.",
			},
			"result_json": schema.StringAttribute{
				Computed:    true,
				Description: "JSON string containing the preview response.",
			},
		},
	}
}

func (d *FormPreviewDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *FormPreviewDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config FormPreviewModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	previewType := "FORM"
	if !config.Type.IsNull() && !config.Type.IsUnknown() && config.Type.ValueString() != "" {
		previewType = config.Type.ValueString()
	}
	body := map[string]interface{}{
		"type":     previewType,
		"template": strings.ToLower(config.Template.ValueString()),
		"content":  config.Content.ValueString(),
	}
	raw, err := d.client.PreviewForm(ctx, config.DomainID.ValueString(), body)
	if err != nil {
		resp.Diagnostics.AddError("Error rendering form preview", err.Error())
		return
	}
	result := formatJSON(raw)

	config.ResultJSON = types.StringValue(result)
	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}

func formatJSON(raw []byte) string {
	if len(raw) == 0 {
		return "null"
	}
	var parsed interface{}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		formatted, _ := json.Marshal(string(raw))
		return string(formatted)
	}
	formatted, _ := json.MarshalIndent(parsed, "", "  ")
	return string(formatted)
}
