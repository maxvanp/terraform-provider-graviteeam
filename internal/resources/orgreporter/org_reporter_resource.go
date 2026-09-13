package orgreporter

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/maxvanp/terraform-provider-graviteeam/internal/client"
)

var (
	_ resource.Resource                = &OrgReporterResource{}
	_ resource.ResourceWithImportState = &OrgReporterResource{}
)

type OrgReporterResource struct {
	client *client.Client
}

func NewOrgReporterResource() resource.Resource {
	return &OrgReporterResource{}
}

func (r *OrgReporterResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_org_reporter"
}

func (r *OrgReporterResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Gravitee AM organization reporter",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the organization reporter",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the organization reporter",
			},
			"type": schema.StringAttribute{
				Required:    true,
				Description: "The reporter plugin type (e.g. reporter-am-file, mongodb)",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"configuration": schema.StringAttribute{
				Required:    true,
				Description: "JSON configuration for the reporter plugin",
			},
			"enabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				Description: "Whether the reporter is enabled",
			},
			"inherited": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Whether the reporter inherits its configuration",
			},
		},
	}
}

func (r *OrgReporterResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *OrgReporterResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan OrgReporterModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.CreateOrgReporter(ctx, buildBody(plan, nil))
	if err != nil {
		resp.Diagnostics.AddError("Error creating organization reporter", err.Error())
		return
	}

	id, err := client.RequiredString(result, "id")
	if err != nil {
		resp.Diagnostics.AddError("Invalid create response", err.Error())
		return
	}
	plan.ID = types.StringValue(id)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *OrgReporterResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state OrgReporterModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.GetOrgReporter(ctx, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading organization reporter", err.Error())
		return
	}

	readIntoModel(&state, result)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *OrgReporterResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan OrgReporterModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state OrgReporterModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.ID = state.ID

	current, err := r.client.GetOrgReporter(ctx, plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading organization reporter before update", err.Error())
		return
	}

	_, err = r.client.UpdateOrgReporter(ctx, plan.ID.ValueString(), buildBody(plan, current))
	if err != nil {
		resp.Diagnostics.AddError("Error updating organization reporter", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *OrgReporterResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state OrgReporterModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteOrgReporter(ctx, state.ID.ValueString())
	if err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Error deleting organization reporter", err.Error())
	}
}

func (r *OrgReporterResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func buildBody(model OrgReporterModel, current map[string]interface{}) map[string]interface{} {
	body := map[string]interface{}{
		"name":          model.Name.ValueString(),
		"type":          model.Type.ValueString(),
		"configuration": model.Configuration.ValueString(),
		"enabled":       model.Enabled.ValueBool(),
		"inherited":     model.Inherited.ValueBool(),
	}
	if model.Inherited.IsUnknown() {
		if inherited, ok := current["inherited"]; ok {
			body["inherited"] = inherited
		}
	}
	return body
}

func readIntoModel(model *OrgReporterModel, data map[string]interface{}) {
	if name, ok := data["name"].(string); ok {
		model.Name = types.StringValue(name)
	}
	if reporterType, ok := data["type"].(string); ok {
		model.Type = types.StringValue(reporterType)
	}
	if enabled, ok := data["enabled"].(bool); ok {
		model.Enabled = types.BoolValue(enabled)
	}
	if inherited, ok := data["inherited"].(bool); ok {
		model.Inherited = types.BoolValue(inherited)
	}
}
