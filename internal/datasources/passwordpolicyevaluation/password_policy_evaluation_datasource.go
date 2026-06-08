package passwordpolicyevaluation

import (
	"context"
	"encoding/json"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/maxvanp/terraform-provider-graviteeam/internal/client"
)

var _ datasource.DataSource = &PasswordPolicyEvaluationDataSource{}

type PasswordPolicyEvaluationDataSource struct {
	client *client.Client
}

type PasswordPolicyEvaluationModel struct {
	DomainID   types.String `tfsdk:"domain_id"`
	PolicyID   types.String `tfsdk:"policy_id"`
	Password   types.String `tfsdk:"password"`
	UserID     types.String `tfsdk:"user_id"`
	ResultJSON types.String `tfsdk:"result_json"`
}

func NewPasswordPolicyEvaluationDataSource() datasource.DataSource {
	return &PasswordPolicyEvaluationDataSource{}
}

func (d *PasswordPolicyEvaluationDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_password_policy_evaluation"
}

func (d *PasswordPolicyEvaluationDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Evaluates a password against a Gravitee AM domain password policy.",
		Attributes: map[string]schema.Attribute{
			"domain_id": schema.StringAttribute{
				Required:    true,
				Description: "The ID of the domain.",
			},
			"policy_id": schema.StringAttribute{
				Required:    true,
				Description: "The ID of the password policy to evaluate. The local 4.11.4 API accepts concrete policy IDs; the default policy alias is not reliable for this endpoint.",
			},
			"password": schema.StringAttribute{
				Required:    true,
				Sensitive:   true,
				Description: "Password value to evaluate.",
			},
			"user_id": schema.StringAttribute{
				Optional:    true,
				Description: "Optional user ID to include in the password evaluation request.",
			},
			"result_json": schema.StringAttribute{
				Computed:    true,
				Description: "JSON string containing the password evaluation response.",
			},
		},
	}
}

func (d *PasswordPolicyEvaluationDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *PasswordPolicyEvaluationDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config PasswordPolicyEvaluationModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]interface{}{
		"password": config.Password.ValueString(),
	}
	if !config.UserID.IsNull() && !config.UserID.IsUnknown() && config.UserID.ValueString() != "" {
		body["userId"] = config.UserID.ValueString()
	}
	raw, err := d.client.EvaluatePasswordPolicy(ctx, config.DomainID.ValueString(), config.PolicyID.ValueString(), body)
	if err != nil {
		resp.Diagnostics.AddError("Error evaluating password policy", err.Error())
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
