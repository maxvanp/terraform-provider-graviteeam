package userrole

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
	_ resource.Resource                = &UserRoleResource{}
	_ resource.ResourceWithImportState = &UserRoleResource{}
)

type UserRoleResource struct {
	client *client.Client
}

func NewUserRoleResource() resource.Resource {
	return &UserRoleResource{}
}

func (r *UserRoleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user_role"
}

func (r *UserRoleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages the set of roles assigned to a Gravitee AM user. Terraform manages the complete role list.",
		Attributes: map[string]schema.Attribute{
			"domain_id": schema.StringAttribute{
				Required:    true,
				Description: "The ID of the domain",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"user_id": schema.StringAttribute{
				Required:    true,
				Description: "The ID of the user",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"roles": schema.SetAttribute{
				Required:    true,
				ElementType: types.StringType,
				Description: "Set of role IDs to assign to the user",
			},
		},
	}
}

func (r *UserRoleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *UserRoleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan UserRoleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	roleIDs := stringValues(plan.Roles)
	if len(roleIDs) > 0 {
		_, err := r.client.SetUserRoles(ctx, plan.DomainID.ValueString(), plan.UserID.ValueString(), roleIDs)
		if err != nil {
			resp.Diagnostics.AddError("Error setting user roles", err.Error())
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *UserRoleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state UserRoleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.GetUserRoles(ctx, state.DomainID.ValueString(), state.UserID.ValueString())
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading user roles", err.Error())
		return
	}

	readIntoModel(&state, result)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *UserRoleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan UserRoleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state UserRoleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domainID := plan.DomainID.ValueString()
	userID := plan.UserID.ValueString()

	toAdd, toRemove := diffRoles(plan.Roles, state.Roles)

	for _, roleID := range toRemove {
		err := r.client.RemoveUserRole(ctx, domainID, userID, roleID)
		if err != nil {
			resp.Diagnostics.AddError("Error removing user role", err.Error())
			return
		}
	}

	if len(toAdd) > 0 {
		_, err := r.client.SetUserRoles(ctx, domainID, userID, toAdd)
		if err != nil {
			resp.Diagnostics.AddError("Error adding user roles", err.Error())
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *UserRoleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state UserRoleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domainID := state.DomainID.ValueString()
	userID := state.UserID.ValueString()

	for _, role := range state.Roles {
		err := r.client.RemoveUserRole(ctx, domainID, userID, role.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Error removing user role", err.Error())
			return
		}
	}
}

func (r *UserRoleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import format: domain_id/user_id
	domainID, userID, ok := parseImportID(req.ID)
	if !ok {
		resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("Expected format: domain_id/user_id, got: %s", req.ID))
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("domain_id"), domainID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("user_id"), userID)...)
}

func parseImportID(id string) (string, string, bool) {
	parts := strings.Split(id, "/")
	if len(parts) != 2 {
		return "", "", false
	}
	return parts[0], parts[1], true
}

func readIntoModel(model *UserRoleModel, roles []interface{}) {
	roleIDs := make([]types.String, 0, len(roles))
	for _, item := range roles {
		roleMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		id, ok := roleMap["id"].(string)
		if !ok {
			continue
		}
		roleIDs = append(roleIDs, types.StringValue(id))
	}
	model.Roles = roleIDs
}

func diffRoles(desired, current []types.String) (toAdd []string, toRemove []string) {
	return reconcile.DiffStrings(stringValues(desired), stringValues(current))
}

func stringValues(values []types.String) []string {
	result := make([]string, len(values))
	for i, value := range values {
		result[i] = value.ValueString()
	}
	return result
}
