package orgrole

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
	_ resource.Resource                = &OrgRoleResource{}
	_ resource.ResourceWithImportState = &OrgRoleResource{}
)

type OrgRoleResource struct {
	client *client.Client
}

func NewOrgRoleResource() resource.Resource {
	return &OrgRoleResource{}
}

func (r *OrgRoleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_org_role"
}

func (r *OrgRoleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Gravitee AM organization role",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the organization role",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the organization role",
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: "The description of the organization role",
			},
			"assignable_type": schema.StringAttribute{
				Required:    true,
				Description: "The assignable type of the role (PLATFORM, DOMAIN, APPLICATION, ORGANIZATION, ENVIRONMENT)",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"permissions": schema.ListAttribute{
				Optional:    true,
				Description: "The list of permissions for this role",
				ElementType: types.StringType,
			},
		},
	}
}

func (r *OrgRoleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *OrgRoleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan OrgRoleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]interface{}{
		"name":           plan.Name.ValueString(),
		"assignableType": plan.AssignableType.ValueString(),
	}
	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		body["description"] = plan.Description.ValueString()
	}

	result, err := r.client.CreateOrgRole(ctx, body)
	if err != nil {
		resp.Diagnostics.AddError("Error creating organization role", err.Error())
		return
	}

	id, err := client.RequiredString(result, "id")
	if err != nil {
		resp.Diagnostics.AddError("Invalid create response", err.Error())
		return
	}
	plan.ID = types.StringValue(id)

	// Create-then-update: if permissions are set, do an update
	hasPermissions := len(plan.Permissions) > 0
	if hasPermissions {
		updateBody := map[string]interface{}{
			"name": plan.Name.ValueString(),
		}
		if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
			updateBody["description"] = plan.Description.ValueString()
		}
		perms := make([]string, len(plan.Permissions))
		for i, p := range plan.Permissions {
			perms[i] = p.ValueString()
		}
		updateBody["permissions"] = perms

		_, err := r.client.UpdateOrgRole(ctx, plan.ID.ValueString(), updateBody)
		if err != nil {
			resp.Diagnostics.AddError("Error updating organization role after creation", err.Error())
			return
		}
	}

	if err := readIntoModel(&plan, result); err != nil {
		resp.Diagnostics.AddError("Invalid create response", err.Error())
		return
	}
	// Re-read to get the full state after potential update
	if hasPermissions {
		readResult, err := r.client.GetOrgRole(ctx, plan.ID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Error reading organization role after creation", err.Error())
			return
		}
		if err := readIntoModel(&plan, readResult); err != nil {
			resp.Diagnostics.AddError("Invalid read response", err.Error())
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *OrgRoleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state OrgRoleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.GetOrgRole(ctx, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading organization role", err.Error())
		return
	}

	if err := readIntoModel(&state, result); err != nil {
		resp.Diagnostics.AddError("Invalid read response", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *OrgRoleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan OrgRoleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state OrgRoleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.ID = state.ID

	body := buildUpdateBody(plan, state)

	_, err := r.client.UpdateOrgRole(ctx, plan.ID.ValueString(), body)
	if err != nil {
		resp.Diagnostics.AddError("Error updating organization role", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *OrgRoleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state OrgRoleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteOrgRole(ctx, state.ID.ValueString())
	if err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Error deleting organization role", err.Error())
	}
}

func (r *OrgRoleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func buildUpdateBody(plan, state OrgRoleModel) map[string]interface{} {
	body := map[string]interface{}{
		"name": plan.Name.ValueString(),
	}
	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		body["description"] = plan.Description.ValueString()
	} else if !state.Description.IsNull() {
		body["description"] = ""
	}
	if plan.Permissions != nil {
		perms := make([]string, len(plan.Permissions))
		for i, p := range plan.Permissions {
			perms[i] = p.ValueString()
		}
		body["permissions"] = perms
	} else if state.Permissions != nil {
		body["permissions"] = []string{}
	}
	return body
}

func readIntoModel(model *OrgRoleModel, result map[string]interface{}) error {
	if name, ok := result["name"].(string); ok {
		model.Name = types.StringValue(name)
	}
	if desc, ok := result["description"].(string); ok && desc != "" {
		model.Description = types.StringValue(desc)
	} else {
		model.Description = types.StringNull()
	}
	if at, ok := result["assignableType"].(string); ok {
		model.AssignableType = types.StringValue(strings.ToUpper(at))
	}
	if perms, ok := result["permissions"].([]interface{}); ok && len(perms) > 0 {
		permissions := make([]types.String, len(perms))
		for i, p := range perms {
			permission, ok := p.(string)
			if !ok {
				return fmt.Errorf("response field %q must contain only strings", "permissions")
			}
			permissions[i] = types.StringValue(permission)
		}
		model.Permissions = permissions
	} else {
		model.Permissions = nil
	}

	return nil
}
