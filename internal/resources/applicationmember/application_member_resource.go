package applicationmember

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
	_ resource.Resource                = &ApplicationMemberResource{}
	_ resource.ResourceWithImportState = &ApplicationMemberResource{}
)

type ApplicationMemberResource struct {
	client *client.Client
}

func NewApplicationMemberResource() resource.Resource {
	return &ApplicationMemberResource{}
}

func (r *ApplicationMemberResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_application_member"
}

func (r *ApplicationMemberResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Gravitee AM application membership role assignment",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the application membership",
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
			"application_id": schema.StringAttribute{
				Required:    true,
				Description: "The ID of the application",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"member_id": schema.StringAttribute{
				Required:    true,
				Description: "The ID of the user or group receiving the application role",
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
				Description: "The ID of the application-assignable organization role to assign",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *ApplicationMemberResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ApplicationMemberResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ApplicationMemberModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]interface{}{
		"memberId":   plan.MemberID.ValueString(),
		"memberType": strings.ToUpper(plan.MemberType.ValueString()),
		"role":       plan.RoleID.ValueString(),
	}
	result, err := r.client.AddOrUpdateApplicationMember(ctx, plan.DomainID.ValueString(), plan.ApplicationID.ValueString(), body)
	if err != nil {
		resp.Diagnostics.AddError("Error creating application member", err.Error())
		return
	}

	if id, ok := result["id"].(string); ok && id != "" {
		plan.ID = types.StringValue(id)
	}
	if err := r.readMembership(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Error reading application member after creation", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ApplicationMemberResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ApplicationMemberModel
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
		resp.Diagnostics.AddError("Error reading application member", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ApplicationMemberResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Unsupported Application Member Update", "Application member attributes require replacement.")
}

func (r *ApplicationMemberResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ApplicationMemberModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteApplicationMember(ctx, state.DomainID.ValueString(), state.ApplicationID.ValueString(), state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Error deleting application member", err.Error())
	}
}

func (r *ApplicationMemberResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import format: domain_id/application_id/member_id/member_type/role_id
	parts := strings.Split(req.ID, "/")
	if len(parts) != 5 {
		resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("Expected format: domain_id/application_id/member_id/member_type/role_id, got: %s", req.ID))
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("domain_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("application_id"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("member_id"), parts[2])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("member_type"), strings.ToUpper(parts[3]))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("role_id"), parts[4])...)
}

func (r *ApplicationMemberResource) readMembership(ctx context.Context, model *ApplicationMemberModel) error {
	memberships, err := r.client.ListApplicationMembers(ctx, model.DomainID.ValueString(), model.ApplicationID.ValueString())
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
	return fmt.Errorf("application member not found: %w", client.ErrNotFound)
}

func matchesMembership(model *ApplicationMemberModel, membership map[string]interface{}) bool {
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

func readIntoModel(model *ApplicationMemberModel, membership map[string]interface{}) {
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
