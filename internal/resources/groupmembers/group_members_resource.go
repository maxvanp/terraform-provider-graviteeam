package groupmembers

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
	"github.com/maxvanp/terraform-provider-graviteeam/internal/resources/importid"
	"github.com/maxvanp/terraform-provider-graviteeam/internal/resources/reconcile"
)

var (
	_ resource.Resource                = &GroupMembersResource{}
	_ resource.ResourceWithImportState = &GroupMembersResource{}
)

type GroupMembersResource struct {
	client *client.Client
}

func NewGroupMembersResource() resource.Resource {
	return &GroupMembersResource{}
}

func (r *GroupMembersResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group_members"
}

func (r *GroupMembersResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages the set of members belonging to a Gravitee AM group. Terraform manages the complete membership list.",
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
			"members": schema.SetAttribute{
				Required:    true,
				ElementType: types.StringType,
				Description: "Set of user IDs that are members of the group",
			},
		},
	}
}

func (r *GroupMembersResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *GroupMembersResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan GroupMembersModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domainID := plan.DomainID.ValueString()
	groupID := plan.GroupID.ValueString()

	for _, member := range plan.Members {
		err := r.client.AddGroupMember(ctx, domainID, groupID, member.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Error adding group member", err.Error())
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *GroupMembersResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state GroupMembersModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	memberIDs, err := r.client.GetGroupMembers(ctx, state.DomainID.ValueString(), state.GroupID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading group members", err.Error())
		return
	}

	readIntoModel(&state, memberIDs)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *GroupMembersResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan GroupMembersModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state GroupMembersModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domainID := plan.DomainID.ValueString()
	groupID := plan.GroupID.ValueString()

	toAdd, toRemove := diffMembers(plan.Members, state.Members)

	for _, memberID := range toRemove {
		err := r.client.RemoveGroupMember(ctx, domainID, groupID, memberID)
		if err != nil {
			resp.Diagnostics.AddError("Error removing group member", err.Error())
			return
		}
	}

	for _, memberID := range toAdd {
		err := r.client.AddGroupMember(ctx, domainID, groupID, memberID)
		if err != nil {
			resp.Diagnostics.AddError("Error adding group member", err.Error())
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *GroupMembersResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state GroupMembersModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domainID := state.DomainID.ValueString()
	groupID := state.GroupID.ValueString()

	for _, member := range state.Members {
		err := r.client.RemoveGroupMember(ctx, domainID, groupID, member.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Error removing group member", err.Error())
			return
		}
	}
}

func (r *GroupMembersResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import format: domain_id/group_id
	domainID, groupID, ok := parseImportID(req.ID)
	if !ok {
		resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("Expected format: domain_id/group_id, got: %s", req.ID))
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("domain_id"), domainID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("group_id"), groupID)...)
}

func parseImportID(id string) (string, string, bool) {
	parts := strings.Split(id, "/")
	if len(parts) != 2 || !importid.Valid(parts) {
		return "", "", false
	}
	return parts[0], parts[1], true
}

func readIntoModel(model *GroupMembersModel, memberIDs []string) {
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
