package orgsettings

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/maxvanp/terraform-provider-graviteeam/internal/client"
)

var (
	_ resource.Resource                = &OrgSettingsResource{}
	_ resource.ResourceWithImportState = &OrgSettingsResource{}
)

type OrgSettingsResource struct {
	client *client.Client
}

func NewOrgSettingsResource() resource.Resource {
	return &OrgSettingsResource{}
}

func (r *OrgSettingsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_org_settings"
}

func (r *OrgSettingsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages Gravitee AM organization settings (singleton resource)",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Stable identifier for the settings resource (always \"settings\")",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"identities": schema.ListAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Description: "List of identity provider IDs to activate at the organization level",
			},
		},
	}
}

func (r *OrgSettingsResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *OrgSettingsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan OrgSettingsModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := buildPatchBody(plan)

	// Only PATCH if there's something to set; an empty PATCH can clear org settings
	if len(body) > 0 {
		_, err := r.client.PatchOrgSettings(ctx, body)
		if err != nil {
			resp.Diagnostics.AddError("Error creating organization settings", err.Error())
			return
		}
	}

	plan.ID = types.StringValue("settings")

	// Read back current state
	result, err := r.client.GetOrgSettings(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error reading organization settings", err.Error())
		return
	}
	readIdentities(&plan, result)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *OrgSettingsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state OrgSettingsModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.GetOrgSettings(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error reading organization settings", err.Error())
		return
	}

	state.ID = types.StringValue("settings")
	readIdentities(&state, result)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *OrgSettingsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan OrgSettingsModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.ID = types.StringValue("settings")

	body := buildPatchBody(plan)

	_, err := r.client.PatchOrgSettings(ctx, body)
	if err != nil {
		resp.Diagnostics.AddError("Error updating organization settings", err.Error())
		return
	}

	// Read back
	result, err := r.client.GetOrgSettings(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error reading organization settings after update", err.Error())
		return
	}
	readIdentities(&plan, result)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *OrgSettingsResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
	// Singleton resource: just remove from Terraform state.
	// We intentionally do NOT clear identities on the server, because
	// clearing the identity provider list would lock out admin access.
}

func (r *OrgSettingsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), "settings")...)
}

func buildPatchBody(model OrgSettingsModel) map[string]interface{} {
	body := map[string]interface{}{}

	if !model.Identities.IsNull() && !model.Identities.IsUnknown() {
		ids := make([]string, 0, len(model.Identities.Elements()))
		for _, v := range model.Identities.Elements() {
			if sv, ok := v.(types.String); ok {
				ids = append(ids, sv.ValueString())
			}
		}
		body["identities"] = ids
	}

	return body
}

func readIdentities(model *OrgSettingsModel, result map[string]interface{}) {
	identitiesRaw, ok := result["identities"]
	if !ok || identitiesRaw == nil {
		model.Identities = types.ListValueMust(types.StringType, []attr.Value{})
		return
	}

	identitiesList, ok := identitiesRaw.([]interface{})
	if !ok {
		model.Identities = types.ListValueMust(types.StringType, []attr.Value{})
		return
	}

	vals := make([]attr.Value, 0, len(identitiesList))
	for _, v := range identitiesList {
		if s, ok := v.(string); ok {
			vals = append(vals, types.StringValue(s))
		}
	}
	model.Identities = types.ListValueMust(types.StringType, vals)
}
