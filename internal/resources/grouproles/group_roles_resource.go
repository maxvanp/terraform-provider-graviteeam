package grouproles

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
	_ resource.Resource                = &GroupRolesResource{}
	_ resource.ResourceWithImportState = &GroupRolesResource{}
)

type GroupRolesResource struct {
	client *client.Client
}

func NewGroupRolesResource() resource.Resource {
	return &GroupRolesResource{}
}

func (r *GroupRolesResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group_roles"
}

func (r *GroupRolesResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages the set of roles assigned to a Gravitee AM group. Terraform manages the complete role list.",
		Attributes: map[string]schema.Attribute{
			"domain_id": schema.StringAttribute{
				Required:    true,
				Description: "The ID of the domain",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"group_id": schema.StringAttribute{
				Required:    true,
				Description: "The ID of the group",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"roles": schema.SetAttribute{
				Required:    true,
				ElementType: types.StringType,
				Description: "Set of role IDs to assign to the group",
			},
		},
	}
}

func (r *GroupRolesResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *GroupRolesResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan GroupRolesModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	roleIDs := make([]string, len(plan.Roles))
	for i, role := range plan.Roles {
		roleIDs[i] = role.ValueString()
	}

	if len(roleIDs) > 0 {
		_, err := r.client.SetGroupRoles(ctx, plan.DomainID.ValueString(), plan.GroupID.ValueString(), roleIDs)
		if err != nil {
			resp.Diagnostics.AddError("Error setting group roles", err.Error())
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *GroupRolesResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state GroupRolesModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.GetGroupRoles(ctx, state.DomainID.ValueString(), state.GroupID.ValueString())
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading group roles", err.Error())
		return
	}

	var roleIDs []types.String
	for _, item := range result {
		if roleMap, ok := item.(map[string]interface{}); ok {
			if id, ok := roleMap["id"].(string); ok {
				roleIDs = append(roleIDs, types.StringValue(id))
			}
		}
	}
	state.Roles = roleIDs

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *GroupRolesResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan GroupRolesModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state GroupRolesModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domainID := plan.DomainID.ValueString()
	groupID := plan.GroupID.ValueString()

	// Build sets for reconciliation
	desired := make(map[string]bool)
	for _, role := range plan.Roles {
		desired[role.ValueString()] = true
	}
	current := make(map[string]bool)
	for _, role := range state.Roles {
		current[role.ValueString()] = true
	}

	// Remove roles no longer desired
	for roleID := range current {
		if !desired[roleID] {
			err := r.client.RemoveGroupRole(ctx, domainID, groupID, roleID)
			if err != nil {
				resp.Diagnostics.AddError("Error removing group role", err.Error())
				return
			}
		}
	}

	// Add new roles
	var toAdd []string
	for roleID := range desired {
		if !current[roleID] {
			toAdd = append(toAdd, roleID)
		}
	}
	if len(toAdd) > 0 {
		_, err := r.client.SetGroupRoles(ctx, domainID, groupID, toAdd)
		if err != nil {
			resp.Diagnostics.AddError("Error adding group roles", err.Error())
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *GroupRolesResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state GroupRolesModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domainID := state.DomainID.ValueString()
	groupID := state.GroupID.ValueString()

	for _, role := range state.Roles {
		err := r.client.RemoveGroupRole(ctx, domainID, groupID, role.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Error removing group role", err.Error())
			return
		}
	}
}

func (r *GroupRolesResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import format: domain_id/group_id
	parts := strings.Split(req.ID, "/")
	if len(parts) != 2 {
		resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("Expected format: domain_id/group_id, got: %s", req.ID))
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("domain_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("group_id"), parts[1])...)
}
