package passwordpolicy

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/maxvanp/terraform-provider-graviteeam/internal/client"
)

var (
	_ resource.Resource                = &PasswordPolicyResource{}
	_ resource.ResourceWithImportState = &PasswordPolicyResource{}
)

type PasswordPolicyResource struct {
	client *client.Client
}

func NewPasswordPolicyResource() resource.Resource {
	return &PasswordPolicyResource{}
}

func (r *PasswordPolicyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_password_policy"
}

func (r *PasswordPolicyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Gravitee AM Password Policy",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the password policy",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"domain_id": schema.StringAttribute{
				Required:    true,
				Description: "The ID of the domain this password policy belongs to",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the password policy",
			},
			"min_length": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "Minimum password length",
			},
			"max_length": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "Maximum password length",
			},
			"max_consecutive_letters": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "Maximum number of consecutive identical characters",
			},
			"expiry_duration": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "Password expiry duration in seconds",
			},
			"old_passwords": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "Number of old passwords to remember",
			},
			"include_numbers": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Require at least one number",
			},
			"include_special_characters": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Require at least one special character",
			},
			"letters_in_mixed_case": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Require both uppercase and lowercase letters",
			},
			"exclude_passwords_in_dictionary": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Reject passwords found in common dictionaries",
			},
			"exclude_user_profile_info_in_password": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Reject passwords containing user profile information",
			},
			"password_history_enabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Enable password history check",
			},
			"default_policy": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether this is the default password policy for the domain",
			},
		},
	}
}

func (r *PasswordPolicyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type", "Expected *client.Client")
		return
	}
	r.client = c
}

func (r *PasswordPolicyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan PasswordPolicyModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Step 1: POST without defaultPolicy (not accepted on create)
	createBody := r.buildBody(plan, false)

	result, err := r.client.CreatePasswordPolicy(ctx, plan.DomainID.ValueString(), createBody)
	if err != nil {
		resp.Diagnostics.AddError("Error creating password policy", err.Error())
		return
	}

	id := result["id"].(string)
	plan.ID = types.StringValue(id)

	// Step 2: PUT with defaultPolicy if set
	if !plan.DefaultPolicy.IsNull() && !plan.DefaultPolicy.IsUnknown() && plan.DefaultPolicy.ValueBool() {
		updateBody := r.buildBody(plan, true)
		result, err = r.client.UpdatePasswordPolicy(ctx, plan.DomainID.ValueString(), id, updateBody)
		if err != nil {
			resp.Diagnostics.AddError("Error updating password policy after creation", err.Error())
			return
		}
	}

	r.readIntoModel(&plan, result)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *PasswordPolicyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state PasswordPolicyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.GetPasswordPolicy(ctx, state.DomainID.ValueString(), state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading password policy", err.Error())
		return
	}

	r.readIntoModel(&state, result)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *PasswordPolicyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan PasswordPolicyModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state PasswordPolicyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.ID = state.ID

	body := r.buildBody(plan, true)

	result, err := r.client.UpdatePasswordPolicy(ctx, plan.DomainID.ValueString(), plan.ID.ValueString(), body)
	if err != nil {
		resp.Diagnostics.AddError("Error updating password policy", err.Error())
		return
	}

	r.readIntoModel(&plan, result)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *PasswordPolicyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state PasswordPolicyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeletePasswordPolicy(ctx, state.DomainID.ValueString(), state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error deleting password policy", err.Error())
	}
}

func (r *PasswordPolicyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 {
		resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("Expected format: domain_id/password_policy_id, got: %s", req.ID))
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("domain_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
}

func (r *PasswordPolicyResource) buildBody(plan PasswordPolicyModel, includeDefaultPolicy bool) map[string]interface{} {
	body := map[string]interface{}{
		"name": plan.Name.ValueString(),
	}

	if !plan.MinLength.IsNull() && !plan.MinLength.IsUnknown() {
		body["minLength"] = plan.MinLength.ValueInt64()
	}
	if !plan.MaxLength.IsNull() && !plan.MaxLength.IsUnknown() {
		body["maxLength"] = plan.MaxLength.ValueInt64()
	}
	if !plan.MaxConsecutiveLetters.IsNull() && !plan.MaxConsecutiveLetters.IsUnknown() {
		body["maxConsecutiveLetters"] = plan.MaxConsecutiveLetters.ValueInt64()
	}
	if !plan.ExpiryDuration.IsNull() && !plan.ExpiryDuration.IsUnknown() {
		body["expiryDuration"] = plan.ExpiryDuration.ValueInt64()
	}
	if !plan.OldPasswords.IsNull() && !plan.OldPasswords.IsUnknown() {
		body["oldPasswords"] = plan.OldPasswords.ValueInt64()
	}
	if !plan.IncludeNumbers.IsNull() && !plan.IncludeNumbers.IsUnknown() {
		body["includeNumbers"] = plan.IncludeNumbers.ValueBool()
	}
	if !plan.IncludeSpecialCharacters.IsNull() && !plan.IncludeSpecialCharacters.IsUnknown() {
		body["includeSpecialCharacters"] = plan.IncludeSpecialCharacters.ValueBool()
	}
	if !plan.LettersInMixedCase.IsNull() && !plan.LettersInMixedCase.IsUnknown() {
		body["lettersInMixedCase"] = plan.LettersInMixedCase.ValueBool()
	}
	if !plan.ExcludePasswordsInDictionary.IsNull() && !plan.ExcludePasswordsInDictionary.IsUnknown() {
		body["excludePasswordsInDictionary"] = plan.ExcludePasswordsInDictionary.ValueBool()
	}
	if !plan.ExcludeUserProfileInfoInPassword.IsNull() && !plan.ExcludeUserProfileInfoInPassword.IsUnknown() {
		body["excludeUserProfileInfoInPassword"] = plan.ExcludeUserProfileInfoInPassword.ValueBool()
	}
	if !plan.PasswordHistoryEnabled.IsNull() && !plan.PasswordHistoryEnabled.IsUnknown() {
		body["passwordHistoryEnabled"] = plan.PasswordHistoryEnabled.ValueBool()
	}
	// defaultPolicy is only accepted on PUT, not POST
	if includeDefaultPolicy && !plan.DefaultPolicy.IsNull() && !plan.DefaultPolicy.IsUnknown() {
		body["defaultPolicy"] = plan.DefaultPolicy.ValueBool()
	}

	return body
}

func (r *PasswordPolicyResource) readIntoModel(model *PasswordPolicyModel, data map[string]interface{}) {
	if id, ok := data["id"].(string); ok {
		model.ID = types.StringValue(id)
	}
	if name, ok := data["name"].(string); ok {
		model.Name = types.StringValue(name)
	}

	model.MinLength = readInt64(data, "minLength")
	model.MaxLength = readInt64(data, "maxLength")
	model.MaxConsecutiveLetters = readInt64(data, "maxConsecutiveLetters")
	model.ExpiryDuration = readInt64(data, "expiryDuration")
	model.OldPasswords = readInt64(data, "oldPasswords")
	model.IncludeNumbers = readBool(data, "includeNumbers")
	model.IncludeSpecialCharacters = readBool(data, "includeSpecialCharacters")
	model.LettersInMixedCase = readBool(data, "lettersInMixedCase")
	model.ExcludePasswordsInDictionary = readBool(data, "excludePasswordsInDictionary")
	model.ExcludeUserProfileInfoInPassword = readBool(data, "excludeUserProfileInfoInPassword")
	model.PasswordHistoryEnabled = readBool(data, "passwordHistoryEnabled")
	model.DefaultPolicy = readBool(data, "defaultPolicy")
}

func readInt64(data map[string]interface{}, key string) types.Int64 {
	if v, ok := data[key]; ok {
		switch n := v.(type) {
		case float64:
			return types.Int64Value(int64(n))
		case int64:
			return types.Int64Value(n)
		}
	}
	return types.Int64Null()
}

func readBool(data map[string]interface{}, key string) types.Bool {
	if v, ok := data[key].(bool); ok {
		return types.BoolValue(v)
	}
	return types.BoolNull()
}
