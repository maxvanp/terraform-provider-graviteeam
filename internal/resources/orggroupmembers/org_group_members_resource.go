package orggroupmembers

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
	"github.com/maxvanp/terraform-provider-graviteeam/internal/resources/reconcile"
)

var (
	_ resource.Resource                = &OrgGroupMembersResource{}
	_ resource.ResourceWithImportState = &OrgGroupMembersResource{}
)

type OrgGroupMembersResource struct {
	client *client.Client
}

func NewOrgGroupMembersResource() resource.Resource {
	return &OrgGroupMembersResource{}
}

func (r *OrgGroupMembersResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_org_group_members"
}

func (r *OrgGroupMembersResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages the complete set of members belonging to a Gravitee AM organization group",
		Attributes: map[string]schema.Attribute{
			"group_id": schema.StringAttribute{
				Required:    true,
				Description: "The ID of the organization group",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"members": schema.SetAttribute{
				Required:    true,
				ElementType: types.StringType,
				Description: "Set of organization user IDs that are members of the group",
			},
		},
	}
}

func (r *OrgGroupMembersResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *OrgGroupMembersResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan OrgGroupMembersModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	for _, member := range plan.Members {
		err := r.client.AddOrgGroupMember(ctx, plan.GroupID.ValueString(), member.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Error adding organization group member", err.Error())
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *OrgGroupMembersResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state OrgGroupMembersModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	memberIDs, err := r.client.GetOrgGroupMembers(ctx, state.GroupID.ValueString())
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading organization group members", err.Error())
		return
	}

	readIntoModel(&state, memberIDs)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *OrgGroupMembersResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan OrgGroupMembersModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state OrgGroupMembersModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	groupID := plan.GroupID.ValueString()
	toAdd, toRemove := diffMembers(plan.Members, state.Members)

	for _, memberID := range toRemove {
		err := r.client.RemoveOrgGroupMember(ctx, groupID, memberID)
		if err != nil {
			resp.Diagnostics.AddError("Error removing organization group member", err.Error())
			return
		}
	}

	for _, memberID := range toAdd {
		err := r.client.AddOrgGroupMember(ctx, groupID, memberID)
		if err != nil {
			resp.Diagnostics.AddError("Error adding organization group member", err.Error())
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *OrgGroupMembersResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state OrgGroupMembersModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	groupID := state.GroupID.ValueString()
	for _, member := range state.Members {
		err := r.client.RemoveOrgGroupMember(ctx, groupID, member.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Error removing organization group member", err.Error())
			return
		}
	}
}

func (r *OrgGroupMembersResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import format: group_id
	if !validImportID(req.ID) {
		resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("Expected format: group_id, got: %s", req.ID))
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("group_id"), req.ID)...)
}

func validImportID(id string) bool {
	return strings.TrimSpace(id) != ""
}

func readIntoModel(model *OrgGroupMembersModel, memberIDs []string) {
	members := make([]types.String, 0, len(memberIDs))
	for _, id := range memberIDs {
		members = append(members, types.StringValue(id))
	}
	model.Members = members
}

func diffMembers(desired, current []types.String) (toAdd []string, toRemove []string) {
	return reconcile.DiffStrings(stringValues(desired), stringValues(current))
}

func stringValues(values []types.String) []string {
	result := make([]string, len(values))
	for i, value := range values {
		result[i] = value.ValueString()
	}
	return result
}
