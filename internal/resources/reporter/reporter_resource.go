package reporter

import (
	"context"
	"fmt"
	"strings"

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
	_ resource.Resource                = &ReporterResource{}
	_ resource.ResourceWithImportState = &ReporterResource{}
)

type ReporterResource struct {
	client *client.Client
}

func NewReporterResource() resource.Resource {
	return &ReporterResource{}
}

func (r *ReporterResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_reporter"
}

func (r *ReporterResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Gravitee AM Reporter (file, jdbc, kafka, mongodb)",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the reporter",
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
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the reporter",
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

func (r *ReporterResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ReporterResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ReporterModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]interface{}{
		"name":          plan.Name.ValueString(),
		"type":          plan.Type.ValueString(),
		"configuration": plan.Configuration.ValueString(),
		"enabled":       plan.Enabled.ValueBool(),
		"inherited":     plan.Inherited.ValueBool(),
	}

	result, err := r.client.CreateReporter(ctx, plan.DomainID.ValueString(), body)
	if err != nil {
		resp.Diagnostics.AddError("Error creating reporter", err.Error())
		return
	}

	plan.ID = types.StringValue(result["id"].(string))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ReporterResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ReporterModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.GetReporter(ctx, state.DomainID.ValueString(), state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading reporter", err.Error())
		return
	}

	if name, ok := result["name"].(string); ok {
		state.Name = types.StringValue(name)
	}
	if t, ok := result["type"].(string); ok {
		state.Type = types.StringValue(t)
	}
	// configuration may contain masked secrets for some reporter types — preserve from state
	if enabled, ok := result["enabled"].(bool); ok {
		state.Enabled = types.BoolValue(enabled)
	}
	if inherited, ok := result["inherited"].(bool); ok {
		state.Inherited = types.BoolValue(inherited)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ReporterResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ReporterModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state ReporterModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.ID = state.ID

	current, err := r.client.GetReporter(ctx, plan.DomainID.ValueString(), plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading reporter before update", err.Error())
		return
	}

	body := buildBody(plan, current)

	_, err = r.client.UpdateReporter(ctx, plan.DomainID.ValueString(), plan.ID.ValueString(), body)
	if err != nil {
		resp.Diagnostics.AddError("Error updating reporter", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func buildBody(plan ReporterModel, current map[string]interface{}) map[string]interface{} {
	body := map[string]interface{}{
		"name":          plan.Name.ValueString(),
		"type":          plan.Type.ValueString(),
		"configuration": plan.Configuration.ValueString(),
		"enabled":       plan.Enabled.ValueBool(),
		"inherited":     plan.Inherited.ValueBool(),
	}
	if plan.Inherited.IsUnknown() {
		if inherited, ok := current["inherited"]; ok {
			body["inherited"] = inherited
		}
	}
	return body
}

func (r *ReporterResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ReporterModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteReporter(ctx, state.DomainID.ValueString(), state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error deleting reporter", err.Error())
	}
}

func (r *ReporterResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 {
		resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("Expected format: domain_id/reporter_id, got: %s", req.ID))
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("domain_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
}
