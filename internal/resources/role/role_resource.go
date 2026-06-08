package role

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
	_ resource.Resource                = &RoleResource{}
	_ resource.ResourceWithImportState = &RoleResource{}
)

type RoleResource struct {
	client *client.Client
}

func NewRoleResource() resource.Resource {
	return &RoleResource{}
}

func (r *RoleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_role"
}

func (r *RoleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Gravitee AM Role",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the role",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"domain_id": schema.StringAttribute{
				Required:    true,
				Description: "The ID of the domain this role belongs to",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the role",
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: "The description of the role",
			},
			"assignable_type": schema.StringAttribute{
				Optional:    true,
				Description: "The assignable type: DOMAIN or APPLICATION",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"permissions": schema.ListAttribute{
				Optional:    true,
				Description: "List of permissions for this role",
				ElementType: types.StringType,
			},
			"oauth_scopes": schema.ListAttribute{
				Optional:    true,
				Description: "List of OAuth2 scopes associated with this role",
				ElementType: types.StringType,
			},
		},
	}
}

func (r *RoleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *RoleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan RoleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Step 1: Create with minimal payload (name, assignableType, description)
	createBody := map[string]interface{}{
		"name": plan.Name.ValueString(),
	}
	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		createBody["description"] = plan.Description.ValueString()
	}
	if !plan.AssignableType.IsNull() && !plan.AssignableType.IsUnknown() {
		createBody["assignableType"] = plan.AssignableType.ValueString()
	}

	result, err := r.client.CreateRole(ctx, plan.DomainID.ValueString(), createBody)
	if err != nil {
		resp.Diagnostics.AddError("Error creating role", err.Error())
		return
	}

	id := result["id"].(string)
	plan.ID = types.StringValue(id)

	// Step 2: Update with full config (permissions, oauthScopes)
	if len(plan.Permissions) > 0 || len(plan.OAuthScopes) > 0 {
		updateBody := r.buildUpdateBody(plan, nil)
		result, err = r.client.UpdateRole(ctx, plan.DomainID.ValueString(), id, updateBody)
		if err != nil {
			resp.Diagnostics.AddError("Error updating role after creation", err.Error())
			return
		}
	}

	r.readIntoModel(&plan, result)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *RoleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state RoleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.GetRole(ctx, state.DomainID.ValueString(), state.ID.ValueString())
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading role", err.Error())
		return
	}

	r.readIntoModel(&state, result)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *RoleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan RoleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state RoleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.ID = state.ID

	updateBody := r.buildUpdateBody(plan, &state)
	result, err := r.client.UpdateRole(ctx, plan.DomainID.ValueString(), plan.ID.ValueString(), updateBody)
	if err != nil {
		resp.Diagnostics.AddError("Error updating role", err.Error())
		return
	}

	r.readIntoModel(&plan, result)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *RoleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state RoleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteRole(ctx, state.DomainID.ValueString(), state.ID.ValueString())
	if err != nil && !strings.Contains(err.Error(), "404") {
		resp.Diagnostics.AddError("Error deleting role", err.Error())
	}
}

func (r *RoleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 {
		resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("Expected format: domain_id/role_id, got: %s", req.ID))
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("domain_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
}

func (r *RoleResource) buildUpdateBody(plan RoleModel, state *RoleModel) map[string]interface{} {
	body := map[string]interface{}{
		"name": plan.Name.ValueString(),
	}

	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		body["description"] = plan.Description.ValueString()
	}

	if plan.Permissions != nil {
		perms := make([]string, len(plan.Permissions))
		for i, p := range plan.Permissions {
			perms[i] = p.ValueString()
		}
		body["permissions"] = perms
	} else if state != nil && state.Permissions != nil {
		body["permissions"] = []string{}
	}

	if plan.OAuthScopes != nil {
		scopes := make([]string, len(plan.OAuthScopes))
		for i, s := range plan.OAuthScopes {
			scopes[i] = s.ValueString()
		}
		body["oauthScopes"] = scopes
	} else if state != nil && state.OAuthScopes != nil {
		body["oauthScopes"] = []string{}
	}

	return body
}

func (r *RoleResource) readIntoModel(model *RoleModel, data map[string]interface{}) {
	if id, ok := data["id"].(string); ok {
		model.ID = types.StringValue(id)
	}
	if name, ok := data["name"].(string); ok {
		model.Name = types.StringValue(name)
	}
	if desc, ok := data["description"].(string); ok {
		model.Description = types.StringValue(desc)
	} else {
		model.Description = types.StringNull()
	}
	if at, ok := data["assignableType"].(string); ok {
		model.AssignableType = types.StringValue(strings.ToUpper(at))
	} else {
		model.AssignableType = types.StringNull()
	}

	if perms, ok := data["permissions"].([]interface{}); ok && len(perms) > 0 {
		model.Permissions = make([]types.String, len(perms))
		for i, p := range perms {
			if s, ok := p.(string); ok {
				model.Permissions[i] = types.StringValue(s)
			}
		}
	}

	if scopes, ok := data["oauthScopes"].([]interface{}); ok && len(scopes) > 0 {
		model.OAuthScopes = make([]types.String, len(scopes))
		for i, s := range scopes {
			if str, ok := s.(string); ok {
				model.OAuthScopes[i] = types.StringValue(str)
			}
		}
	}
}
