package orgentrypoint

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/maxvanp/terraform-provider-graviteeam/internal/client"
)

var (
	_ resource.Resource                = &OrgEntrypointResource{}
	_ resource.ResourceWithImportState = &OrgEntrypointResource{}
)

type OrgEntrypointResource struct {
	client *client.Client
}

func NewOrgEntrypointResource() resource.Resource {
	return &OrgEntrypointResource{}
}

func (r *OrgEntrypointResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_org_entrypoint"
}

func (r *OrgEntrypointResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Gravitee AM organization entrypoint",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the organization entrypoint",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the organization entrypoint",
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: "The description of the organization entrypoint",
			},
			"url": schema.StringAttribute{
				Required:    true,
				Description: "The URL of the organization entrypoint",
			},
			"tags": schema.ListAttribute{
				Required:    true,
				Description: "List of organization tag IDs associated with this entrypoint",
				ElementType: types.StringType,
			},
			"default_entrypoint": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether this is the default entrypoint",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *OrgEntrypointResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *OrgEntrypointResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan OrgEntrypointModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.CreateOrgEntrypoint(ctx, buildBody(plan))
	if err != nil {
		resp.Diagnostics.AddError("Error creating organization entrypoint", err.Error())
		return
	}

	readIntoModel(&plan, result)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *OrgEntrypointResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state OrgEntrypointModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.GetOrgEntrypoint(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading organization entrypoint", err.Error())
		return
	}

	readIntoModel(&state, result)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *OrgEntrypointResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan OrgEntrypointModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state OrgEntrypointModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.ID = state.ID

	result, err := r.client.UpdateOrgEntrypoint(ctx, plan.ID.ValueString(), buildBody(plan, state))
	if err != nil {
		resp.Diagnostics.AddError("Error updating organization entrypoint", err.Error())
		return
	}

	readIntoModel(&plan, result)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *OrgEntrypointResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state OrgEntrypointModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteOrgEntrypoint(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error deleting organization entrypoint", err.Error())
	}
}

func (r *OrgEntrypointResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func buildBody(model OrgEntrypointModel, state ...OrgEntrypointModel) map[string]interface{} {
	body := map[string]interface{}{
		"name": model.Name.ValueString(),
		"url":  model.URL.ValueString(),
		"tags": stringValues(model.Tags),
	}
	if !model.Description.IsNull() && !model.Description.IsUnknown() {
		body["description"] = model.Description.ValueString()
	} else if len(state) > 0 && !state[0].Description.IsNull() {
		body["description"] = ""
	}
	return body
}

func readIntoModel(model *OrgEntrypointModel, result map[string]interface{}) {
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
	if url, ok := result["url"].(string); ok {
		model.URL = types.StringValue(url)
	}
	if defaultEntrypoint, ok := result["defaultEntrypoint"].(bool); ok {
		model.DefaultEntrypoint = types.BoolValue(defaultEntrypoint)
	} else {
		model.DefaultEntrypoint = types.BoolNull()
	}
	if tags, ok := result["tags"].([]interface{}); ok {
		model.Tags = interfaceStrings(tags)
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
