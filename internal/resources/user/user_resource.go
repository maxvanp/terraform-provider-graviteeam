package user

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/maxvanp/terraform-provider-graviteeam/internal/client"
	"github.com/maxvanp/terraform-provider-graviteeam/internal/resources/importid"
)

var (
	_ resource.Resource                = &UserResource{}
	_ resource.ResourceWithImportState = &UserResource{}
)

type UserResource struct {
	client *client.Client
}

func NewUserResource() resource.Resource {
	return &UserResource{}
}

func (r *UserResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user"
}

func (r *UserResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Gravitee AM User",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the user",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"domain_id": schema.StringAttribute{
				Required:    true,
				Description: "The ID of the domain this user belongs to",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"username": schema.StringAttribute{
				Required:    true,
				Description: "The username",
			},
			"email": schema.StringAttribute{
				Optional:    true,
				Description: "The email address",
			},
			"first_name": schema.StringAttribute{
				Optional:    true,
				Description: "The first name",
			},
			"last_name": schema.StringAttribute{
				Optional:    true,
				Description: "The last name",
			},
			"display_name": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The display name",
			},
			"force_reset_password": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Whether the user must reset their password at next login",
			},
			"enabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Whether the user is enabled",
			},
			"locked": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Whether the user account is locked",
			},
			"pre_registration": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				Description: "Whether this is a pre-registration (user must set password). Defaults to true.",
			},
			"reset_password": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "Password value to send to the Gravitee AM reset password endpoint when reset_password_trigger changes.",
			},
			"reset_password_trigger": schema.StringAttribute{
				Optional:    true,
				Description: "Arbitrary value used to explicitly reset the user password. Changing this value calls the Gravitee AM reset password endpoint with reset_password.",
			},
			"registration_confirmation_trigger": schema.StringAttribute{
				Optional:    true,
				Description: "Arbitrary value used to explicitly send the user registration confirmation email. Changing this value calls the Gravitee AM send registration confirmation endpoint.",
			},
		},
	}
}

func (r *UserResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *UserResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan UserModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]interface{}{
		"username":        plan.Username.ValueString(),
		"enabled":         plan.Enabled.ValueBool(),
		"preRegistration": plan.PreRegistration.ValueBool(),
	}
	if !plan.Email.IsNull() {
		body["email"] = plan.Email.ValueString()
	}
	if !plan.FirstName.IsNull() {
		body["firstName"] = plan.FirstName.ValueString()
	}
	if !plan.LastName.IsNull() {
		body["lastName"] = plan.LastName.ValueString()
	}
	body["forceResetPassword"] = plan.ForceResetPassword.ValueBool()

	result, err := r.client.CreateUser(ctx, plan.DomainID.ValueString(), body)
	if err != nil {
		resp.Diagnostics.AddError("Error creating user", err.Error())
		return
	}

	id, err := client.RequiredString(result, "id")
	if err != nil {
		resp.Diagnostics.AddError("Invalid create response", err.Error())
		return
	}
	plan.ID = types.StringValue(id)
	desiredLocked := plan.Locked.ValueBool()
	if !plan.DisplayName.IsNull() && !plan.DisplayName.IsUnknown() {
		current, err := r.client.GetUser(ctx, plan.DomainID.ValueString(), plan.ID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Error reading user before post-create update", err.Error())
			return
		}
		body := buildMergedUpdateBody(current, plan)
		result, err = r.client.UpdateUser(ctx, plan.DomainID.ValueString(), plan.ID.ValueString(), body)
		if err != nil {
			resp.Diagnostics.AddError("Error updating user after creation", err.Error())
			return
		}
		r.readIntoModel(&plan, result)
		plan.Locked = types.BoolValue(desiredLocked)
	}
	if plan.Locked.ValueBool() {
		if err := r.client.LockUser(ctx, plan.DomainID.ValueString(), plan.ID.ValueString()); err != nil {
			resp.Diagnostics.AddError("Error locking user", err.Error())
			return
		}
	}

	result, err = r.client.GetUser(ctx, plan.DomainID.ValueString(), plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading user after creation", err.Error())
		return
	}
	r.readIntoModel(&plan, result)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *UserResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state UserModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.GetUser(ctx, state.DomainID.ValueString(), state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading user", err.Error())
		return
	}

	r.readIntoModel(&state, result)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *UserResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan UserModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state UserModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.ID = state.ID
	plannedResetPassword := plan.ResetPassword

	if plan.Username.ValueString() != state.Username.ValueString() {
		_, err := r.client.UpdateUsername(ctx, plan.DomainID.ValueString(), plan.ID.ValueString(), plan.Username.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Error updating username", err.Error())
			return
		}
	}

	profileChanged := !plan.Email.Equal(state.Email) ||
		!plan.FirstName.Equal(state.FirstName) ||
		!plan.LastName.Equal(state.LastName) ||
		!plan.DisplayName.Equal(state.DisplayName) ||
		plan.ForceResetPassword.ValueBool() != state.ForceResetPassword.ValueBool() ||
		plan.PreRegistration.ValueBool() != state.PreRegistration.ValueBool()

	if profileChanged {
		current, err := r.client.GetUser(ctx, plan.DomainID.ValueString(), plan.ID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Error reading user before update", err.Error())
			return
		}
		body := buildMergedUpdateBody(current, plan)
		_, err = r.client.UpdateUser(ctx, plan.DomainID.ValueString(), plan.ID.ValueString(), body)
		if err != nil {
			resp.Diagnostics.AddError("Error updating user", err.Error())
			return
		}
	}

	if plan.Enabled.ValueBool() != state.Enabled.ValueBool() {
		_, err := r.client.UpdateUserStatus(ctx, plan.DomainID.ValueString(), plan.ID.ValueString(), plan.Enabled.ValueBool())
		if err != nil {
			resp.Diagnostics.AddError("Error updating user status", err.Error())
			return
		}
	}

	if plan.Locked.ValueBool() != state.Locked.ValueBool() {
		if plan.Locked.ValueBool() {
			err := r.client.LockUser(ctx, plan.DomainID.ValueString(), plan.ID.ValueString())
			if err != nil {
				resp.Diagnostics.AddError("Error locking user", err.Error())
				return
			}
		} else {
			err := r.client.UnlockUser(ctx, plan.DomainID.ValueString(), plan.ID.ValueString())
			if err != nil {
				resp.Diagnostics.AddError("Error unlocking user", err.Error())
				return
			}
		}
	}

	if shouldResetPassword(plan.ResetTrigger, state.ResetTrigger) {
		if plan.ResetPassword.IsNull() || plan.ResetPassword.IsUnknown() || plan.ResetPassword.ValueString() == "" {
			resp.Diagnostics.AddAttributeError(
				path.Root("reset_password"),
				"Missing Reset Password",
				"reset_password must be set when reset_password_trigger is changed.",
			)
			return
		}
		if err := r.client.ResetUserPassword(ctx, plan.DomainID.ValueString(), plan.ID.ValueString(), plan.ResetPassword.ValueString()); err != nil {
			resp.Diagnostics.AddError("Error resetting user password", err.Error())
			return
		}
	}

	if shouldTrigger(plan.RegistrationTrigger, state.RegistrationTrigger) {
		if err := r.client.SendUserRegistrationConfirmation(ctx, plan.DomainID.ValueString(), plan.ID.ValueString()); err != nil {
			resp.Diagnostics.AddError("Error sending user registration confirmation", err.Error())
			return
		}
	}

	result, err := r.client.GetUser(ctx, plan.DomainID.ValueString(), plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading user after update", err.Error())
		return
	}
	r.readIntoModel(&plan, result)
	plan.ResetPassword = plannedResetPassword
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *UserResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state UserModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteUser(ctx, state.DomainID.ValueString(), state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Error deleting user", err.Error())
	}
}

func shouldResetPassword(planTrigger, stateTrigger types.String) bool {
	return shouldTrigger(planTrigger, stateTrigger)
}

func shouldTrigger(planTrigger, stateTrigger types.String) bool {
	if planTrigger.IsNull() || planTrigger.IsUnknown() {
		return false
	}
	if stateTrigger.IsNull() || stateTrigger.IsUnknown() {
		return true
	}
	return planTrigger.ValueString() != stateTrigger.ValueString()
}

func (r *UserResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import format: domain_id/user_id
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 || !importid.Valid(parts) {
		resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("Expected format: domain_id/user_id, got: %s", req.ID))
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("domain_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
}

func (r *UserResource) readIntoModel(model *UserModel, data map[string]interface{}) {
	if id, ok := data["id"].(string); ok {
		model.ID = types.StringValue(id)
	}
	if username, ok := data["username"].(string); ok {
		model.Username = types.StringValue(username)
	}
	if email, ok := data["email"].(string); ok {
		model.Email = types.StringValue(email)
	} else {
		model.Email = types.StringNull()
	}
	if firstName, ok := data["firstName"].(string); ok {
		model.FirstName = types.StringValue(firstName)
	} else {
		model.FirstName = types.StringNull()
	}
	if lastName, ok := data["lastName"].(string); ok {
		model.LastName = types.StringValue(lastName)
	} else {
		model.LastName = types.StringNull()
	}
	if displayName, ok := data["displayName"].(string); ok {
		model.DisplayName = types.StringValue(displayName)
	} else {
		model.DisplayName = types.StringNull()
	}
	if forceResetPassword, ok := data["forceResetPassword"].(bool); ok {
		model.ForceResetPassword = types.BoolValue(forceResetPassword)
	}
	if enabled, ok := data["enabled"].(bool); ok {
		model.Enabled = types.BoolValue(enabled)
	}
	if accountNonLocked, ok := data["accountNonLocked"].(bool); ok {
		model.Locked = types.BoolValue(!accountNonLocked)
	}
	if preReg, ok := data["preRegistration"].(bool); ok {
		model.PreRegistration = types.BoolValue(preReg)
	}
}

func buildMergedUpdateBody(current map[string]interface{}, plan UserModel) map[string]interface{} {
	body := copyUserUpdateFields(current)
	if !plan.Email.IsNull() && !plan.Email.IsUnknown() {
		body["email"] = plan.Email.ValueString()
	}
	if !plan.FirstName.IsNull() && !plan.FirstName.IsUnknown() {
		body["firstName"] = plan.FirstName.ValueString()
	}
	if !plan.LastName.IsNull() && !plan.LastName.IsUnknown() {
		body["lastName"] = plan.LastName.ValueString()
	}
	if !plan.DisplayName.IsNull() && !plan.DisplayName.IsUnknown() {
		body["displayName"] = plan.DisplayName.ValueString()
	}
	body["forceResetPassword"] = plan.ForceResetPassword.ValueBool()
	body["preRegistration"] = plan.PreRegistration.ValueBool()
	return body
}

func copyUserUpdateFields(current map[string]interface{}) map[string]interface{} {
	allowed := []string{
		"accountNonExpired",
		"accountNonLocked",
		"additionalInformation",
		"client",
		"createdAt",
		"credentialsNonExpired",
		"displayName",
		"email",
		"enabled",
		"externalId",
		"firstName",
		"forceResetPassword",
		"lastName",
		"loggedAt",
		"loginsCount",
		"preRegistration",
		"preferredLanguage",
		"registrationCompleted",
		"source",
		"updatedAt",
	}
	body := make(map[string]interface{}, len(allowed))
	for _, field := range allowed {
		if value, ok := current[field]; ok {
			body[field] = value
		}
	}
	return body
}
