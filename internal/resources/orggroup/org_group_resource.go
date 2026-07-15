package orggroup

import (
	"context"
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
	_ resource.Resource                = &OrgGroupResource{}
	_ resource.ResourceWithImportState = &OrgGroupResource{}
)

type OrgGroupResource struct {
	client *client.Client
}

func NewOrgGroupResource() resource.Resource {
	return &OrgGroupResource{}
}

func (r *OrgGroupResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_org_group"
}

func (r *OrgGroupResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Gravitee AM organization group",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the organization group",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the organization group",
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: "The description of the organization group",
			},
			"members": schema.ListAttribute{
				Optional:    true,
				Description: "List of organization user IDs that are members of this group. Do not manage this attribute together with graviteeam_org_group_members for the same group.",
				ElementType: types.StringType,
			},
			"roles": schema.ListAttribute{
				Optional:    true,
				Description: "List of organization role IDs assigned to this group. Use either this inline attribute or separate membership resources for a given group collection, not both.",
				ElementType: types.StringType,
			},
		},
	}
}

func (r *OrgGroupResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *OrgGroupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan OrgGroupModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createBody := map[string]interface{}{
		"name": plan.Name.ValueString(),
	}
	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		createBody["description"] = plan.Description.ValueString()
	}
	if plan.Members != nil {
		createBody["members"] = stringValues(plan.Members)
	}

	result, err := r.client.CreateOrgGroup(ctx, createBody)
	if err != nil {
		resp.Diagnostics.AddError("Error creating organization group", err.Error())
		return
	}

	plan.ID = types.StringValue(result["id"].(string))

	if plan.Roles != nil {
		updateBody := buildBody(plan, OrgGroupModel{})
		result, err = r.client.UpdateOrgGroup(ctx, plan.ID.ValueString(), updateBody)
		if err != nil {
			resp.Diagnostics.AddError("Error updating organization group after creation", err.Error())
			return
		}
	}

	savedMembers := plan.Members
	savedRoles := plan.Roles
	readIntoModel(&plan, result)
	if savedMembers != nil {
		plan.Members = savedMembers
	}
	if savedRoles != nil {
		plan.Roles = savedRoles
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *OrgGroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state OrgGroupModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.GetOrgGroup(ctx, state.ID.ValueString())
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading organization group", err.Error())
		return
	}

	savedMembers := state.Members
	savedRoles := state.Roles
	readIntoModel(&state, result)
	if savedMembers != nil && state.Members == nil {
		state.Members = savedMembers
	} else if savedMembers == nil {
		state.Members = nil
	}
	if savedRoles != nil && state.Roles == nil {
		state.Roles = savedRoles
	} else if savedRoles == nil {
		state.Roles = nil
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *OrgGroupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan OrgGroupModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state OrgGroupModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.ID = state.ID

	result, err := r.client.UpdateOrgGroup(ctx, plan.ID.ValueString(), buildBody(plan, state))
	if err != nil {
		resp.Diagnostics.AddError("Error updating organization group", err.Error())
		return
	}

	savedMembers := plan.Members
	savedRoles := plan.Roles
	readIntoModel(&plan, result)
	if savedMembers != nil {
		plan.Members = savedMembers
	}
	if savedRoles != nil {
		plan.Roles = savedRoles
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *OrgGroupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state OrgGroupModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteOrgGroup(ctx, state.ID.ValueString())
	if err != nil && !strings.Contains(err.Error(), "404") {
		resp.Diagnostics.AddError("Error deleting organization group", err.Error())
	}
}

func (r *OrgGroupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func buildBody(model, state OrgGroupModel) map[string]interface{} {
	body := map[string]interface{}{
		"name": model.Name.ValueString(),
	}
	if !model.Description.IsNull() && !model.Description.IsUnknown() {
		body["description"] = model.Description.ValueString()
	} else if !state.Description.IsNull() {
		body["description"] = ""
	}
	if model.Members != nil {
		body["members"] = stringValues(model.Members)
	} else if state.Members != nil {
		body["members"] = []string{}
	}
	if model.Roles != nil {
		body["roles"] = stringValues(model.Roles)
	} else if state.Roles != nil {
		body["roles"] = []string{}
	}
	return body
}

func readIntoModel(model *OrgGroupModel, result map[string]interface{}) {
	if id, ok := result["id"].(string); ok {
		model.ID = types.StringValue(id)
	}
	if name, ok := result["name"].(string); ok {
		model.Name = types.StringValue(name)
	}
	if desc, ok := result["description"].(string); ok && desc != "" {
		model.Description = types.StringValue(desc)
	} else {
		model.Description = types.StringNull()
	}
	if members, ok := result["members"].([]interface{}); ok && len(members) > 0 {
		model.Members = interfaceStrings(members)
	} else {
		model.Members = nil
	}
	if roles, ok := result["roles"].([]interface{}); ok && len(roles) > 0 {
		model.Roles = interfaceStrings(roles)
	} else {
		model.Roles = nil
	}
}

func stringValues(values []types.String) []string {
	result := make([]string, len(values))
	for i, value := range values {
		result[i] = value.ValueString()
	}
	return result
}

func interfaceStrings(values []interface{}) []types.String {
	result := make([]types.String, len(values))
	for i, value := range values {
		if stringValue, ok := value.(string); ok {
			result[i] = types.StringValue(stringValue)
		}
	}
	return result
}
