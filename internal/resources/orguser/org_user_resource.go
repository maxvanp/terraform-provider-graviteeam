package orguser

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/maxvanp/terraform-provider-graviteeam/internal/client"
)

var (
	_ resource.Resource                = &OrgUserResource{}
	_ resource.ResourceWithImportState = &OrgUserResource{}
)

type OrgUserResource struct {
	client *client.Client
}

func NewOrgUserResource() resource.Resource {
	return &OrgUserResource{}
}

func (r *OrgUserResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_org_user"
}

func (r *OrgUserResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Gravitee AM organization user",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the organization user",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"username": schema.StringAttribute{
				Required:    true,
				Description: "The username",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"password": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "The initial password. Required when creating an organization user.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
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
			"enabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				Description: "Whether the user is enabled",
			},
			"pre_registration": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				Description: "Whether this is a pre-registration user",
			},
		},
	}
}

func (r *OrgUserResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *OrgUserResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan OrgUserModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if plan.Password.IsNull() || plan.Password.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("password"),
			"Missing Organization User Password",
			"The Gravitee AM Management API requires an initial password when creating an organization user.",
		)
		return
	}

	desiredEnabled := plan.Enabled.ValueBool()
	result, err := r.client.CreateOrgUser(ctx, buildCreateBody(plan))
	if err != nil {
		resp.Diagnostics.AddError("Error creating organization user", err.Error())
		return
	}

	readIntoModel(&plan, result)
	if desiredEnabled != plan.Enabled.ValueBool() {
		result, err = r.client.UpdateOrgUserStatus(ctx, plan.ID.ValueString(), desiredEnabled)
		if err != nil {
			resp.Diagnostics.AddError("Error updating organization user status after creation", err.Error())
			return
		}
		readIntoModel(&plan, result)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *OrgUserResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state OrgUserModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.GetOrgUser(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading organization user", err.Error())
		return
	}

	readIntoModel(&state, result)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *OrgUserResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan OrgUserModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state OrgUserModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.ID = state.ID

	if updateBody := buildUpdateBody(plan); len(updateBody) > 0 {
		_, err := r.client.UpdateOrgUser(ctx, plan.ID.ValueString(), updateBody)
		if err != nil {
			resp.Diagnostics.AddError("Error updating organization user", err.Error())
			return
		}
	}

	if plan.Enabled.ValueBool() != state.Enabled.ValueBool() {
		_, err := r.client.UpdateOrgUserStatus(ctx, plan.ID.ValueString(), plan.Enabled.ValueBool())
		if err != nil {
			resp.Diagnostics.AddError("Error updating organization user status", err.Error())
			return
		}
	}

	result, err := r.client.GetOrgUser(ctx, plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading organization user after update", err.Error())
		return
	}
	readIntoModel(&plan, result)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *OrgUserResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state OrgUserModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteOrgUser(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error deleting organization user", err.Error())
	}
}

func (r *OrgUserResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func buildCreateBody(model OrgUserModel) map[string]interface{} {
	body := buildUpdateBody(model)
	body["username"] = model.Username.ValueString()
	if !model.Password.IsNull() && !model.Password.IsUnknown() {
		body["password"] = model.Password.ValueString()
	}
	return body
}

func buildUpdateBody(model OrgUserModel) map[string]interface{} {
	body := map[string]interface{}{
		"preRegistration": model.PreRegistration.ValueBool(),
	}
	if !model.Email.IsNull() && !model.Email.IsUnknown() {
		body["email"] = model.Email.ValueString()
	}
	if !model.FirstName.IsNull() && !model.FirstName.IsUnknown() {
		body["firstName"] = model.FirstName.ValueString()
	}
	if !model.LastName.IsNull() && !model.LastName.IsUnknown() {
		body["lastName"] = model.LastName.ValueString()
	}
	return body
}

func readIntoModel(model *OrgUserModel, data map[string]interface{}) {
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
	if enabled, ok := data["enabled"].(bool); ok {
		model.Enabled = types.BoolValue(enabled)
	}
	if preRegistration, ok := data["preRegistration"].(bool); ok {
		model.PreRegistration = types.BoolValue(preRegistration)
	}
}
