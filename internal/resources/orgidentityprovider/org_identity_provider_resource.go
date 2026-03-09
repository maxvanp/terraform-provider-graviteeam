package orgidentityprovider

import (
	"context"
	"encoding/json"

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
	_ resource.Resource                = &OrgIdentityProviderResource{}
	_ resource.ResourceWithImportState = &OrgIdentityProviderResource{}
)

type OrgIdentityProviderResource struct {
	client *client.Client
}

func NewOrgIdentityProviderResource() resource.Resource {
	return &OrgIdentityProviderResource{}
}

func (r *OrgIdentityProviderResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_org_identity_provider"
}

func (r *OrgIdentityProviderResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Gravitee AM organization-level identity provider",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the organization identity provider",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the organization identity provider",
			},
			"type": schema.StringAttribute{
				Required:    true,
				Description: "The type of identity provider (e.g. inline-am-idp)",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"configuration": schema.StringAttribute{
				Required:    true,
				Sensitive:   true,
				Description: "JSON configuration for the identity provider (use jsonencode())",
			},
			"domain_whitelist": schema.ListAttribute{
				Optional:    true,
				Description: "List of whitelisted email domains",
				ElementType: types.StringType,
			},
			"external": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Whether this is an external/social identity provider",
			},
		},
	}
}

func (r *OrgIdentityProviderResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *OrgIdentityProviderResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan OrgIdentityProviderModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]interface{}{
		"name":          plan.Name.ValueString(),
		"type":          plan.Type.ValueString(),
		"configuration": plan.Configuration.ValueString(),
		"external":      plan.External.ValueBool(),
	}

	if plan.DomainWhitelist != nil {
		wl := make([]string, len(plan.DomainWhitelist))
		for i, d := range plan.DomainWhitelist {
			wl[i] = d.ValueString()
		}
		body["domainWhitelist"] = wl
	}

	result, err := r.client.CreateOrgIdentityProvider(ctx, body)
	if err != nil {
		resp.Diagnostics.AddError("Error creating organization identity provider", err.Error())
		return
	}

	plan.ID = types.StringValue(result["id"].(string))

	// Preserve configuration from plan (API may mask sensitive values)
	savedConfig := plan.Configuration
	readIntoModel(&plan, result)
	plan.Configuration = savedConfig
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *OrgIdentityProviderResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state OrgIdentityProviderModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.GetOrgIdentityProvider(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading organization identity provider", err.Error())
		return
	}

	// Preserve configuration from state (API may mask sensitive values)
	savedConfig := state.Configuration
	readIntoModel(&state, result)
	state.Configuration = savedConfig
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *OrgIdentityProviderResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan OrgIdentityProviderModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state OrgIdentityProviderModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.ID = state.ID

	body := map[string]interface{}{
		"name":          plan.Name.ValueString(),
		"configuration": plan.Configuration.ValueString(),
	}

	if plan.DomainWhitelist != nil {
		wl := make([]string, len(plan.DomainWhitelist))
		for i, d := range plan.DomainWhitelist {
			wl[i] = d.ValueString()
		}
		body["domainWhitelist"] = wl
	}

	result, err := r.client.UpdateOrgIdentityProvider(ctx, plan.ID.ValueString(), body)
	if err != nil {
		resp.Diagnostics.AddError("Error updating organization identity provider", err.Error())
		return
	}

	// Preserve configuration from plan (API may mask sensitive values)
	savedConfig := plan.Configuration
	readIntoModel(&plan, result)
	plan.Configuration = savedConfig
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *OrgIdentityProviderResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state OrgIdentityProviderModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteOrgIdentityProvider(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error deleting organization identity provider", err.Error())
	}
}

func (r *OrgIdentityProviderResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func readIntoModel(model *OrgIdentityProviderModel, data map[string]interface{}) {
	if id, ok := data["id"].(string); ok {
		model.ID = types.StringValue(id)
	}
	if name, ok := data["name"].(string); ok {
		model.Name = types.StringValue(name)
	}
	if t, ok := data["type"].(string); ok {
		model.Type = types.StringValue(t)
	}
	if ext, ok := data["external"].(bool); ok {
		model.External = types.BoolValue(ext)
	}

	// Configuration can be a string or a map
	if cfg, ok := data["configuration"].(string); ok {
		model.Configuration = types.StringValue(cfg)
	} else if cfg, ok := data["configuration"].(map[string]interface{}); ok {
		cfgBytes, err := json.Marshal(cfg)
		if err == nil {
			model.Configuration = types.StringValue(string(cfgBytes))
		}
	}

	// Domain whitelist
	if wl, ok := data["domainWhitelist"].([]interface{}); ok && len(wl) > 0 {
		model.DomainWhitelist = make([]types.String, len(wl))
		for i, d := range wl {
			if s, ok := d.(string); ok {
				model.DomainWhitelist[i] = types.StringValue(s)
			}
		}
	}
}
