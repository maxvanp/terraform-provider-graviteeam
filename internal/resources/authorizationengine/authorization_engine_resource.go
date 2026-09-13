package authorizationengine

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
	_ resource.Resource                = &AuthorizationEngineResource{}
	_ resource.ResourceWithImportState = &AuthorizationEngineResource{}
)

type AuthorizationEngineResource struct {
	client *client.Client
}

func NewAuthorizationEngineResource() resource.Resource {
	return &AuthorizationEngineResource{}
}

func (r *AuthorizationEngineResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_authorization_engine"
}

func (r *AuthorizationEngineResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Gravitee AM authorization engine plugin",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the authorization engine",
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
				Description: "The name of the authorization engine",
			},
			"type": schema.StringAttribute{
				Required:    true,
				Description: "The authorization engine plugin type",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"configuration": schema.StringAttribute{
				Required:    true,
				Description: "JSON configuration for the authorization engine plugin",
			},
		},
	}
}

func (r *AuthorizationEngineResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *AuthorizationEngineResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan AuthorizationEngineModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := buildBody(plan)

	result, err := r.client.CreateAuthorizationEngine(ctx, plan.DomainID.ValueString(), body)
	if err != nil {
		resp.Diagnostics.AddError("Error creating authorization engine", err.Error())
		return
	}

	readIntoModel(&plan, result)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *AuthorizationEngineResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state AuthorizationEngineModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.GetAuthorizationEngine(ctx, state.DomainID.ValueString(), state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading authorization engine", err.Error())
		return
	}

	readIntoModel(&state, result)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *AuthorizationEngineResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan AuthorizationEngineModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state AuthorizationEngineModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.ID = state.ID
	body := buildBody(plan)

	result, err := r.client.UpdateAuthorizationEngine(ctx, plan.DomainID.ValueString(), plan.ID.ValueString(), body)
	if err != nil {
		resp.Diagnostics.AddError("Error updating authorization engine", err.Error())
		return
	}

	readIntoModel(&plan, result)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *AuthorizationEngineResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state AuthorizationEngineModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteAuthorizationEngine(ctx, state.DomainID.ValueString(), state.ID.ValueString())
	if err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Error deleting authorization engine", err.Error())
	}
}

func (r *AuthorizationEngineResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	domainID, engineID, ok := parseImportID(req.ID)
	if !ok {
		resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("Expected format: domain_id/authorization_engine_id, got: %s", req.ID))
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("domain_id"), domainID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), engineID)...)
}

func parseImportID(id string) (string, string, bool) {
	parts := strings.SplitN(id, "/", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	return parts[0], parts[1], true
}

func buildBody(plan AuthorizationEngineModel) map[string]interface{} {
	return map[string]interface{}{
		"name":          plan.Name.ValueString(),
		"type":          plan.Type.ValueString(),
		"configuration": plan.Configuration.ValueString(),
	}
}

func readIntoModel(model *AuthorizationEngineModel, data map[string]interface{}) {
	if id, ok := data["id"].(string); ok {
		model.ID = types.StringValue(id)
	}
	if name, ok := data["name"].(string); ok {
		model.Name = types.StringValue(name)
	}
	if t, ok := data["type"].(string); ok {
		model.Type = types.StringValue(t)
	}
	if configuration, ok := data["configuration"].(string); ok {
		model.Configuration = types.StringValue(configuration)
	}
}
