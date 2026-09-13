package scope

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
	"github.com/maxvanp/terraform-provider-graviteeam/internal/resources/importid"
)

var (
	_ resource.Resource                = &ScopeResource{}
	_ resource.ResourceWithImportState = &ScopeResource{}
)

type ScopeResource struct {
	client *client.Client
}

func NewScopeResource() resource.Resource {
	return &ScopeResource{}
}

func (r *ScopeResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_scope"
}

func (r *ScopeResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Gravitee AM OAuth2 Scope",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the scope",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"domain_id": schema.StringAttribute{
				Required:    true,
				Description: "The ID of the domain this scope belongs to",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"key": schema.StringAttribute{
				Required:    true,
				Description: "The scope key (e.g. 'roles', 'custom_claim')",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The display name of the scope",
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: "The description of the scope",
			},
			"discovery": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				Description: "Whether the scope is visible in the discovery endpoint",
			},
			"expires_in": schema.Int64Attribute{
				Optional:    true,
				Description: "Scope expiration in seconds",
			},
			"icon_uri": schema.StringAttribute{
				Optional:    true,
				Description: "URI of the icon associated with the scope",
			},
			"parameterized": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Whether the scope is parameterized",
			},
		},
	}
}

func (r *ScopeResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ScopeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ScopeModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := r.buildBody(plan)

	result, err := r.client.CreateScope(ctx, plan.DomainID.ValueString(), body)
	if err != nil {
		resp.Diagnostics.AddError("Error creating scope", err.Error())
		return
	}

	id, err := client.RequiredString(result, "id")
	if err != nil {
		resp.Diagnostics.AddError("Invalid create response", err.Error())
		return
	}
	plan.ID = types.StringValue(id)

	r.readIntoModel(&plan, result)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ScopeResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ScopeModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.GetScope(ctx, state.DomainID.ValueString(), state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading scope", err.Error())
		return
	}

	r.readIntoModel(&state, result)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ScopeResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ScopeModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state ScopeModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.ID = state.ID

	body := r.buildUpdateBody(plan, state)

	result, err := r.client.UpdateScope(ctx, plan.DomainID.ValueString(), plan.ID.ValueString(), body)
	if err != nil {
		resp.Diagnostics.AddError("Error updating scope", err.Error())
		return
	}

	r.readIntoModel(&plan, result)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ScopeResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ScopeModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteScope(ctx, state.DomainID.ValueString(), state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Error deleting scope", err.Error())
	}
}

func (r *ScopeResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 || !importid.Valid(parts) {
		resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("Expected format: domain_id/scope_id, got: %s", req.ID))
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("domain_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
}

func (r *ScopeResource) buildBody(plan ScopeModel) map[string]interface{} {
	body := map[string]interface{}{
		"key":           plan.Key.ValueString(),
		"name":          plan.Name.ValueString(),
		"discovery":     plan.Discovery.ValueBool(),
		"parameterized": plan.Parameterized.ValueBool(),
	}

	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		body["description"] = plan.Description.ValueString()
	}
	if !plan.ExpiresIn.IsNull() && !plan.ExpiresIn.IsUnknown() {
		body["expiresIn"] = plan.ExpiresIn.ValueInt64()
	}
	if !plan.IconURI.IsNull() && !plan.IconURI.IsUnknown() {
		body["iconUri"] = plan.IconURI.ValueString()
	}

	return body
}

func (r *ScopeResource) buildUpdateBody(plan, state ScopeModel) map[string]interface{} {
	body := map[string]interface{}{
		"name":          plan.Name.ValueString(),
		"discovery":     plan.Discovery.ValueBool(),
		"parameterized": plan.Parameterized.ValueBool(),
	}

	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		body["description"] = plan.Description.ValueString()
	} else if !state.Description.IsNull() {
		body["description"] = ""
	}
	if !plan.ExpiresIn.IsNull() && !plan.ExpiresIn.IsUnknown() {
		body["expiresIn"] = plan.ExpiresIn.ValueInt64()
	} else if !state.ExpiresIn.IsNull() {
		body["expiresIn"] = 0
	}
	if !plan.IconURI.IsNull() && !plan.IconURI.IsUnknown() {
		body["iconUri"] = plan.IconURI.ValueString()
	} else if !state.IconURI.IsNull() {
		body["iconUri"] = nil
	}

	return body
}

func (r *ScopeResource) readIntoModel(model *ScopeModel, data map[string]interface{}) {
	if id, ok := data["id"].(string); ok {
		model.ID = types.StringValue(id)
	}
	if key, ok := data["key"].(string); ok {
		model.Key = types.StringValue(key)
	}
	if name, ok := data["name"].(string); ok {
		model.Name = types.StringValue(name)
	}
	if desc, ok := data["description"].(string); ok && desc != "" {
		model.Description = types.StringValue(desc)
	} else {
		model.Description = types.StringNull()
	}
	if disc, ok := data["discovery"].(bool); ok {
		model.Discovery = types.BoolValue(disc)
	}
	if v, ok := data["expiresIn"]; ok {
		switch n := v.(type) {
		case float64:
			if n == 0 && model.ExpiresIn.IsNull() {
				model.ExpiresIn = types.Int64Null()
			} else {
				model.ExpiresIn = types.Int64Value(int64(n))
			}
		case int64:
			if n == 0 && model.ExpiresIn.IsNull() {
				model.ExpiresIn = types.Int64Null()
			} else {
				model.ExpiresIn = types.Int64Value(n)
			}
		}
	} else {
		model.ExpiresIn = types.Int64Null()
	}
	if iconURI, ok := data["iconUri"].(string); ok && iconURI != "" {
		model.IconURI = types.StringValue(iconURI)
	} else {
		model.IconURI = types.StringNull()
	}
	if parameterized, ok := data["parameterized"].(bool); ok {
		model.Parameterized = types.BoolValue(parameterized)
	}
}
