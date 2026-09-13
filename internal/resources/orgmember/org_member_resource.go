package orgmember

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
)

var (
	_ resource.Resource                = &OrgMemberResource{}
	_ resource.ResourceWithImportState = &OrgMemberResource{}
)

type OrgMemberResource struct {
	client *client.Client
}

func NewOrgMemberResource() resource.Resource {
	return &OrgMemberResource{}
}

func (r *OrgMemberResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_org_member"
}

func (r *OrgMemberResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Gravitee AM organization membership role assignment",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the organization membership",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"member_id": schema.StringAttribute{
				Required:    true,
				Description: "The ID of the user or group receiving the organization role",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"member_type": schema.StringAttribute{
				Required:    true,
				Description: "The member type (USER or GROUP)",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"role_id": schema.StringAttribute{
				Required:    true,
				Description: "The ID of the organization role to assign",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *OrgMemberResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *OrgMemberResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan OrgMemberModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := buildBody(plan)
	result, err := r.client.AddOrUpdateOrgMember(ctx, body)
	if err != nil {
		resp.Diagnostics.AddError("Error creating organization member", err.Error())
		return
	}

	if id, ok := result["id"].(string); ok && id != "" {
		plan.ID = types.StringValue(id)
	}
	if err := r.readMembership(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Error reading organization member after creation", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *OrgMemberResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state OrgMemberModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.readMembership(ctx, &state)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading organization member", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *OrgMemberResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Unsupported Organization Member Update", "Organization member attributes require replacement.")
}

func (r *OrgMemberResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state OrgMemberModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteOrgMember(ctx, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Error deleting organization member", err.Error())
	}
}

func (r *OrgMemberResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import format: member_id/member_type/role_id
	memberID, memberType, roleID, ok := parseImportID(req.ID)
	if !ok {
		resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("Expected format: member_id/member_type/role_id, got: %s", req.ID))
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("member_id"), memberID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("member_type"), memberType)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("role_id"), roleID)...)
}

func parseImportID(id string) (string, string, string, bool) {
	parts := strings.Split(id, "/")
	if len(parts) != 3 || !importid.Valid(parts) {
		return "", "", "", false
	}
	return parts[0], strings.ToUpper(parts[1]), parts[2], true
}

func buildBody(plan OrgMemberModel) map[string]interface{} {
	return map[string]interface{}{
		"memberId":   plan.MemberID.ValueString(),
		"memberType": strings.ToUpper(plan.MemberType.ValueString()),
		"role":       plan.RoleID.ValueString(),
	}
}

func (r *OrgMemberResource) readMembership(ctx context.Context, model *OrgMemberModel) error {
	memberships, err := r.client.ListOrgMembers(ctx)
	if err != nil {
		return err
	}
	for _, membership := range memberships {
		if !matchesMembership(model, membership) {
			continue
		}
		readIntoModel(model, membership)
		return nil
	}
	return fmt.Errorf("organization member not found: %w", client.ErrNotFound)
}

func matchesMembership(model *OrgMemberModel, membership map[string]interface{}) bool {
	if !model.ID.IsNull() && !model.ID.IsUnknown() && model.ID.ValueString() != "" {
		if id, ok := membership["id"].(string); ok && id == model.ID.ValueString() {
			return true
		}
	}
	memberID, _ := membership["memberId"].(string)
	memberType, _ := membership["memberType"].(string)
	roleID, _ := membership["roleId"].(string)
	return memberID == model.MemberID.ValueString() &&
		strings.EqualFold(memberType, model.MemberType.ValueString()) &&
		roleID == model.RoleID.ValueString()
}

func readIntoModel(model *OrgMemberModel, membership map[string]interface{}) {
	if id, ok := membership["id"].(string); ok {
		model.ID = types.StringValue(id)
	}
	if memberID, ok := membership["memberId"].(string); ok {
		model.MemberID = types.StringValue(memberID)
	}
	if memberType, ok := membership["memberType"].(string); ok {
		model.MemberType = types.StringValue(strings.ToUpper(memberType))
	}
	if roleID, ok := membership["roleId"].(string); ok {
		model.RoleID = types.StringValue(roleID)
	}
}
