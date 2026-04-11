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
			"pre_registration": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				Description: "Whether this is a pre-registration (user must set password). Defaults to true.",
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

	result, err := r.client.CreateUser(ctx, plan.DomainID.ValueString(), body)
	if err != nil {
		resp.Diagnostics.AddError("Error creating user", err.Error())
		return
	}

	plan.ID = types.StringValue(result["id"].(string))
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

	body := map[string]interface{}{}
	if !plan.Email.IsNull() {
		body["email"] = plan.Email.ValueString()
	}
	if !plan.FirstName.IsNull() {
		body["firstName"] = plan.FirstName.ValueString()
	}
	if !plan.LastName.IsNull() {
		body["lastName"] = plan.LastName.ValueString()
	}

	_, err := r.client.UpdateUser(ctx, plan.DomainID.ValueString(), plan.ID.ValueString(), body)
	if err != nil {
		resp.Diagnostics.AddError("Error updating user", err.Error())
		return
	}

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
		resp.Diagnostics.AddError("Error deleting user", err.Error())
	}
}

func (r *UserResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import format: domain_id/user_id
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 {
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
	if preReg, ok := data["preRegistration"].(bool); ok {
		model.PreRegistration = types.BoolValue(preReg)
	}
}
