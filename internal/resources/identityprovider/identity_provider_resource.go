package identityprovider

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
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
	_ resource.Resource                = &IdentityProviderResource{}
	_ resource.ResourceWithImportState = &IdentityProviderResource{}
)

type IdentityProviderResource struct {
	client *client.Client
}

func NewIdentityProviderResource() resource.Resource {
	return &IdentityProviderResource{}
}

func (r *IdentityProviderResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_identity_provider"
}

func (r *IdentityProviderResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Gravitee AM Identity Provider",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the identity provider",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"domain_id": schema.StringAttribute{
				Required:    true,
				Description: "The ID of the domain this identity provider belongs to",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the identity provider",
			},
			"type": schema.StringAttribute{
				Required:    true,
				Description: "The type of identity provider (e.g. inline-am-idp, http-am-idp, jdbc-am-idp, mongo-am-idp, oauth2-generic-am-idp)",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"external": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Whether this is an external/social identity provider (shows a separate login button). Can only be set at creation time.",
			},
			"configuration": schema.StringAttribute{
				Required:    true,
				Description: "JSON configuration for the identity provider (use jsonencode())",
			},
			"mappers": schema.MapAttribute{
				Optional:    true,
				Description: "Attribute mapping (e.g. username = \"uid\", email = \"mail\")",
				ElementType: types.StringType,
			},
			"domain_whitelist": schema.ListAttribute{
				Optional:    true,
				Description: "List of whitelisted email domains",
				ElementType: types.StringType,
			},
			"password_policy_id": schema.StringAttribute{
				Optional:    true,
				Description: "The ID of the password policy to associate with this identity provider",
			},
			"group_mapper": schema.MapAttribute{
				Optional:    true,
				Description: "Group mapping rules: key is an EL condition (e.g. \"{true}\"), value is a list of group IDs",
				ElementType: types.ListType{ElemType: types.StringType},
			},
			"role_mapper": schema.MapAttribute{
				Optional:    true,
				Description: "Role mapping rules: key is an EL condition (e.g. \"{true}\"), value is a list of role IDs",
				ElementType: types.ListType{ElemType: types.StringType},
			},
		},
	}
}

func (r *IdentityProviderResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *IdentityProviderResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan IdentityProviderModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Step 1: Create with name, type, configuration, external
	createBody := map[string]interface{}{
		"name":          plan.Name.ValueString(),
		"type":          plan.Type.ValueString(),
		"configuration": plan.Configuration.ValueString(),
		"external":      plan.External.ValueBool(),
	}

	result, err := r.client.CreateIdentityProvider(ctx, plan.DomainID.ValueString(), createBody)
	if err != nil {
		resp.Diagnostics.AddError("Error creating identity provider", err.Error())
		return
	}

	id := result["id"].(string)
	plan.ID = types.StringValue(id)

	// Step 2: Update with full config (mappers, domainWhitelist, passwordPolicy, groupMapper, roleMapper)
	needsUpdate := len(plan.Mappers) > 0 ||
		len(plan.DomainWhitelist) > 0 ||
		(!plan.PasswordPolicyID.IsNull() && !plan.PasswordPolicyID.IsUnknown()) ||
		(!plan.GroupMapper.IsNull() && !plan.GroupMapper.IsUnknown()) ||
		(!plan.RoleMapper.IsNull() && !plan.RoleMapper.IsUnknown())

	if needsUpdate {
		updateBody := r.buildUpdateBody(plan)
		result, err = r.client.UpdateIdentityProvider(ctx, plan.DomainID.ValueString(), id, updateBody)
		if err != nil {
			resp.Diagnostics.AddError("Error updating identity provider after creation", err.Error())
			return
		}
	}

	// Preserve plan values for fields the API masks or doesn't return
	savedConfig := plan.Configuration
	savedPasswordPolicy := plan.PasswordPolicyID
	r.readIntoModel(&plan, result)
	plan.Configuration = savedConfig
	plan.PasswordPolicyID = savedPasswordPolicy
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *IdentityProviderResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state IdentityProviderModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.GetIdentityProvider(ctx, state.DomainID.ValueString(), state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading identity provider", err.Error())
		return
	}

	// Preserve state values for fields the API masks or doesn't return
	savedConfig := state.Configuration
	savedPasswordPolicy := state.PasswordPolicyID
	r.readIntoModel(&state, result)
	state.Configuration = savedConfig
	state.PasswordPolicyID = savedPasswordPolicy
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *IdentityProviderResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan IdentityProviderModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state IdentityProviderModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.ID = state.ID

	updateBody := r.buildUpdateBody(plan)
	result, err := r.client.UpdateIdentityProvider(ctx, plan.DomainID.ValueString(), plan.ID.ValueString(), updateBody)
	if err != nil {
		resp.Diagnostics.AddError("Error updating identity provider", err.Error())
		return
	}

	// Preserve plan values for fields the API masks or doesn't return
	savedConfig := plan.Configuration
	savedPasswordPolicy := plan.PasswordPolicyID
	r.readIntoModel(&plan, result)
	plan.Configuration = savedConfig
	plan.PasswordPolicyID = savedPasswordPolicy
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *IdentityProviderResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state IdentityProviderModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteIdentityProvider(ctx, state.DomainID.ValueString(), state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error deleting identity provider", err.Error())
	}
}

func (r *IdentityProviderResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 {
		resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("Expected format: domain_id/identity_provider_id, got: %s", req.ID))
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("domain_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
}

func (r *IdentityProviderResource) buildUpdateBody(plan IdentityProviderModel) map[string]interface{} {
	body := map[string]interface{}{
		"name":          plan.Name.ValueString(),
		"type":          plan.Type.ValueString(),
		"configuration": plan.Configuration.ValueString(),
	}

	if len(plan.Mappers) > 0 {
		mappers := make(map[string]string, len(plan.Mappers))
		for k, v := range plan.Mappers {
			mappers[k] = v.ValueString()
		}
		body["mappers"] = mappers
	}

	if plan.DomainWhitelist != nil {
		wl := make([]string, len(plan.DomainWhitelist))
		for i, d := range plan.DomainWhitelist {
			wl[i] = d.ValueString()
		}
		body["domainWhitelist"] = wl
	}

	if !plan.PasswordPolicyID.IsNull() && !plan.PasswordPolicyID.IsUnknown() {
		body["passwordPolicy"] = plan.PasswordPolicyID.ValueString()
	}

	if !plan.GroupMapper.IsNull() && !plan.GroupMapper.IsUnknown() {
		body["groupMapper"] = invertConditionMapperToAPI(plan.GroupMapper)
	}

	if !plan.RoleMapper.IsNull() && !plan.RoleMapper.IsUnknown() {
		body["roleMapper"] = invertConditionMapperToAPI(plan.RoleMapper)
	}

	return body
}

func (r *IdentityProviderResource) readIntoModel(model *IdentityProviderModel, data map[string]interface{}) {
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

	// Mappers
	if mappers, ok := data["mappers"].(map[string]interface{}); ok && len(mappers) > 0 {
		model.Mappers = make(map[string]types.String, len(mappers))
		for k, v := range mappers {
			if s, ok := v.(string); ok {
				model.Mappers[k] = types.StringValue(s)
			}
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

	// Password policy
	if pp, ok := data["passwordPolicy"].(string); ok && pp != "" {
		model.PasswordPolicyID = types.StringValue(pp)
	} else {
		model.PasswordPolicyID = types.StringNull()
	}

	// API format is targetId => [conditions], HCL format is condition => [targetIds].
	listType := types.ListType{ElemType: types.StringType}
	model.GroupMapper = readAPIConditionMapper(data, "groupMapper", listType)
	model.RoleMapper = readAPIConditionMapper(data, "roleMapper", listType)
}

func invertConditionMapperToAPI(tfMap types.Map) map[string][]string {
	apiMapper := make(map[string][]string)
	for condition, v := range tfMap.Elements() {
		listVal, ok := v.(types.List)
		if !ok {
			continue
		}
		for _, elem := range listVal.Elements() {
			if s, ok := elem.(types.String); ok {
				id := s.ValueString()
				apiMapper[id] = append(apiMapper[id], condition)
			}
		}
	}
	return apiMapper
}

func readAPIConditionMapper(data map[string]interface{}, key string, listType types.ListType) types.Map {
	mapper, ok := data[key].(map[string]interface{})
	if !ok || len(mapper) == 0 {
		return types.MapNull(listType)
	}

	inverted := make(map[string][]string)
	for id, v := range mapper {
		if conditions, ok := v.([]interface{}); ok {
			for _, c := range conditions {
				if cond, ok := c.(string); ok {
					inverted[cond] = append(inverted[cond], id)
				}
			}
		}
	}

	elems := make(map[string]attr.Value, len(inverted))
	for condition, ids := range inverted {
		values := make([]attr.Value, len(ids))
		for i, id := range ids {
			values[i] = types.StringValue(id)
		}
		listVal, _ := types.ListValue(types.StringType, values)
		elems[condition] = listVal
	}

	mapVal, _ := types.MapValue(listType, elems)
	return mapVal
}
