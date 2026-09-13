package domainmember

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
	_ resource.Resource                = &DomainMemberResource{}
	_ resource.ResourceWithImportState = &DomainMemberResource{}
)

type DomainMemberResource struct {
	client *client.Client
}

func NewDomainMemberResource() resource.Resource {
	return &DomainMemberResource{}
}

func (r *DomainMemberResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_domain_member"
}

func (r *DomainMemberResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Gravitee AM domain membership role assignment",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the domain membership",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"domain_id": schema.StringAttribute{
				Required:    true,
				Description: "The ID of the domain",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"member_id": schema.StringAttribute{
				Required:    true,
				Description: "The ID of the user or group receiving the domain role",
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
				Description: "The ID of the domain-assignable organization role to assign",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *DomainMemberResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *DomainMemberResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan DomainMemberModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := buildBody(plan)
	result, err := r.client.AddOrUpdateDomainMember(ctx, plan.DomainID.ValueString(), body)
	if err != nil {
		resp.Diagnostics.AddError("Error creating domain member", err.Error())
		return
	}

	if id, ok := result["id"].(string); ok && id != "" {
		plan.ID = types.StringValue(id)
	}
	if err := r.readMembership(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Error reading domain member after creation", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *DomainMemberResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state DomainMemberModel
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
		resp.Diagnostics.AddError("Error reading domain member", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *DomainMemberResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Unsupported Domain Member Update", "Domain member attributes require replacement.")
}

func (r *DomainMemberResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state DomainMemberModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteDomainMember(ctx, state.DomainID.ValueString(), state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Error deleting domain member", err.Error())
	}
}

func (r *DomainMemberResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import format: domain_id/member_id/member_type/role_id
	domainID, memberID, memberType, roleID, ok := parseImportID(req.ID)
	if !ok {
		resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("Expected format: domain_id/member_id/member_type/role_id, got: %s", req.ID))
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("domain_id"), domainID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("member_id"), memberID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("member_type"), memberType)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("role_id"), roleID)...)
}

func parseImportID(id string) (string, string, string, string, bool) {
	parts := strings.Split(id, "/")
	if len(parts) != 4 {
		return "", "", "", "", false
	}
	return parts[0], parts[1], strings.ToUpper(parts[2]), parts[3], true
}

func buildBody(plan DomainMemberModel) map[string]interface{} {
	return map[string]interface{}{
		"memberId":   plan.MemberID.ValueString(),
		"memberType": strings.ToUpper(plan.MemberType.ValueString()),
		"role":       plan.RoleID.ValueString(),
	}
}

func (r *DomainMemberResource) readMembership(ctx context.Context, model *DomainMemberModel) error {
	memberships, err := r.client.ListDomainMembers(ctx, model.DomainID.ValueString())
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
	return fmt.Errorf("domain member not found: %w", client.ErrNotFound)
}

func matchesMembership(model *DomainMemberModel, membership map[string]interface{}) bool {
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

func readIntoModel(model *DomainMemberModel, membership map[string]interface{}) {
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
