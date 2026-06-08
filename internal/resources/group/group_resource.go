package group

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
	_ resource.Resource                = &GroupResource{}
	_ resource.ResourceWithImportState = &GroupResource{}
)

type GroupResource struct {
	client *client.Client
}

func NewGroupResource() resource.Resource {
	return &GroupResource{}
}

func (r *GroupResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group"
}

func (r *GroupResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Gravitee AM Group",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the group",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"domain_id": schema.StringAttribute{
				Required:    true,
				Description: "The ID of the domain this group belongs to",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the group",
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: "The description of the group",
			},
			"members": schema.ListAttribute{
				Optional:    true,
				Description: "List of user IDs that are members of this group. Do not manage this attribute together with graviteeam_group_members for the same group.",
				ElementType: types.StringType,
			},
			"roles": schema.ListAttribute{
				Optional:    true,
				Description: "List of role IDs assigned to this group. Do not manage this attribute together with graviteeam_group_roles for the same group.",
				ElementType: types.StringType,
			},
		},
	}
}

func (r *GroupResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *GroupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan GroupModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Step 1: Create with name, description, members
	createBody := map[string]interface{}{
		"name": plan.Name.ValueString(),
	}
	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		createBody["description"] = plan.Description.ValueString()
	}
	if len(plan.Members) > 0 {
		members := make([]string, len(plan.Members))
		for i, m := range plan.Members {
			members[i] = m.ValueString()
		}
		createBody["members"] = members
	}

	result, err := r.client.CreateGroup(ctx, plan.DomainID.ValueString(), createBody)
	if err != nil {
		resp.Diagnostics.AddError("Error creating group", err.Error())
		return
	}

	id := result["id"].(string)
	plan.ID = types.StringValue(id)

	// Step 2: Update with roles (create-then-update pattern)
	if len(plan.Roles) > 0 {
		updateBody := r.buildUpdateBody(plan, nil)
		result, err = r.client.UpdateGroup(ctx, plan.DomainID.ValueString(), id, updateBody)
		if err != nil {
			resp.Diagnostics.AddError("Error updating group after creation", err.Error())
			return
		}
	}

	// Preserve plan values for fields the API may not return
	savedRoles := plan.Roles
	savedMembers := plan.Members
	r.readIntoModel(&plan, result)
	if savedRoles != nil {
		plan.Roles = savedRoles
	}
	if savedMembers != nil {
		plan.Members = savedMembers
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *GroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state GroupModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.GetGroup(ctx, state.DomainID.ValueString(), state.ID.ValueString())
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading group", err.Error())
		return
	}

	// Preserve state consistency for roles and members
	// If state had roles/members, preserve them (API may not return them)
	// If state didn't have roles/members, don't import them from API
	// (they may be managed by graviteeam_group_roles or graviteeam_group_members)
	savedRoles := state.Roles
	savedMembers := state.Members
	r.readIntoModel(&state, result)
	if savedRoles != nil && state.Roles == nil {
		state.Roles = savedRoles
	} else if savedRoles == nil {
		state.Roles = nil
	}
	if savedMembers != nil && state.Members == nil {
		state.Members = savedMembers
	} else if savedMembers == nil {
		state.Members = nil
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *GroupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan GroupModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state GroupModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.ID = state.ID

	updateBody := r.buildUpdateBody(plan, &state)
	result, err := r.client.UpdateGroup(ctx, plan.DomainID.ValueString(), plan.ID.ValueString(), updateBody)
	if err != nil {
		resp.Diagnostics.AddError("Error updating group", err.Error())
		return
	}

	// Preserve plan values for fields the API may not return
	savedRoles := plan.Roles
	savedMembers := plan.Members
	r.readIntoModel(&plan, result)
	if savedRoles != nil {
		plan.Roles = savedRoles
	}
	if savedMembers != nil {
		plan.Members = savedMembers
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *GroupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state GroupModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteGroup(ctx, state.DomainID.ValueString(), state.ID.ValueString())
	if err != nil && !strings.Contains(err.Error(), "404") {
		resp.Diagnostics.AddError("Error deleting group", err.Error())
	}
}

func (r *GroupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 {
		resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("Expected format: domain_id/group_id, got: %s", req.ID))
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("domain_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
}

func (r *GroupResource) buildUpdateBody(plan GroupModel, state *GroupModel) map[string]interface{} {
	body := map[string]interface{}{
		"name": plan.Name.ValueString(),
	}

	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		body["description"] = plan.Description.ValueString()
	}

	if plan.Members != nil {
		members := make([]string, len(plan.Members))
		for i, m := range plan.Members {
			members[i] = m.ValueString()
		}
		body["members"] = members
	} else if state != nil && state.Members != nil {
		body["members"] = []string{}
	}

	if plan.Roles != nil {
		roles := make([]string, len(plan.Roles))
		for i, r := range plan.Roles {
			roles[i] = r.ValueString()
		}
		body["roles"] = roles
	} else if state != nil && state.Roles != nil {
		body["roles"] = []string{}
	}

	return body
}

func (r *GroupResource) readIntoModel(model *GroupModel, data map[string]interface{}) {
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

	if members, ok := data["members"].([]interface{}); ok && len(members) > 0 {
		model.Members = make([]types.String, len(members))
		for i, m := range members {
			if s, ok := m.(string); ok {
				model.Members[i] = types.StringValue(s)
			}
		}
	} else {
		model.Members = nil
	}

	if roles, ok := data["roles"].([]interface{}); ok && len(roles) > 0 {
		model.Roles = make([]types.String, len(roles))
		for i, r := range roles {
			if s, ok := r.(string); ok {
				model.Roles[i] = types.StringValue(s)
			}
		}
	} else {
		model.Roles = nil
	}
}
