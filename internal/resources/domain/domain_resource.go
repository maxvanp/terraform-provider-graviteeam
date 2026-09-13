package domain

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/maxvanp/terraform-provider-graviteeam/internal/client"
)

var (
	_ resource.Resource                = &DomainResource{}
	_ resource.ResourceWithImportState = &DomainResource{}
)

type DomainResource struct {
	client *client.Client
}

func NewDomainResource() resource.Resource {
	return &DomainResource{}
}

func (r *DomainResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_domain"
}

func (r *DomainResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Gravitee AM Security Domain",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the domain",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the domain",
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: "The description of the domain",
			},
			"enabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Whether the domain is enabled",
			},
			"data_plane_id": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("default"),
				Description: "The data plane ID used when creating the domain",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"default_idp_id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the default identity provider (default-idp-{domain_id})",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"settings_json": schema.StringAttribute{
				Optional:    true,
				Description: "JSON object for advanced domain patch settings. The value is merged into the Gravitee AM domain payload; typed attributes and blocks override matching keys.",
			},
		},
		Blocks: map[string]schema.Block{
			"oidc": schema.SingleNestedBlock{
				Description: "OIDC client registration settings",
				Attributes: map[string]schema.Attribute{
					"allow_localhost_redirect_uri": schema.BoolAttribute{
						Optional:    true,
						Computed:    true,
						Default:     booldefault.StaticBool(false),
						Description: "Allow localhost redirect URIs",
					},
					"allow_http_scheme_redirect_uri": schema.BoolAttribute{
						Optional:    true,
						Computed:    true,
						Default:     booldefault.StaticBool(false),
						Description: "Allow HTTP scheme redirect URIs",
					},
					"allow_wildcard_redirect_uri": schema.BoolAttribute{
						Optional:    true,
						Computed:    true,
						Default:     booldefault.StaticBool(false),
						Description: "Allow wildcard redirect URIs",
					},
					"dynamic_client_registration_enabled": schema.BoolAttribute{
						Optional:    true,
						Computed:    true,
						Default:     booldefault.StaticBool(false),
						Description: "Enable dynamic client registration",
					},
				},
			},
			"login_settings": schema.SingleNestedBlock{
				Description: "Login page settings",
				Attributes: map[string]schema.Attribute{
					"register_enabled": schema.BoolAttribute{
						Optional:    true,
						Computed:    true,
						Default:     booldefault.StaticBool(false),
						Description: "Enable self-registration",
					},
					"forgot_password_enabled": schema.BoolAttribute{
						Optional:    true,
						Computed:    true,
						Default:     booldefault.StaticBool(false),
						Description: "Enable forgot password",
					},
					"identifier_first_enabled": schema.BoolAttribute{
						Optional:    true,
						Computed:    true,
						Default:     booldefault.StaticBool(false),
						Description: "Enable identifier-first login (username only, then route to IdP)",
					},
				},
			},
		},
	}
}

func (r *DomainResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *DomainResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan DomainModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Step 1: Create with minimal payload (name + description only)
	createBody := map[string]interface{}{
		"name":        plan.Name.ValueString(),
		"dataPlaneId": plan.DataPlaneID.ValueString(),
	}
	if !plan.Description.IsNull() {
		createBody["description"] = plan.Description.ValueString()
	}

	result, err := r.client.CreateDomain(ctx, createBody)
	if err != nil {
		resp.Diagnostics.AddError("Error creating domain", err.Error())
		return
	}

	id, err := client.RequiredString(result, "id")
	if err != nil {
		resp.Diagnostics.AddError("Invalid create response", err.Error())
		return
	}
	plan.ID = types.StringValue(id)
	plan.DefaultIdpID = types.StringValue("default-idp-" + id)

	// Step 2: Update immediately with full config (enabled, oidc, loginSettings)
	updateBody, err := r.buildUpdateBody(plan, nil)
	if err != nil {
		resp.Diagnostics.AddError("Invalid domain configuration", err.Error())
		return
	}
	result, err = r.client.UpdateDomain(ctx, id, updateBody)
	if err != nil {
		resp.Diagnostics.AddError("Error updating domain after creation", err.Error())
		return
	}

	r.readIntoModel(&plan, result)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *DomainResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state DomainModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.GetDomain(ctx, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading domain", err.Error())
		return
	}

	r.readIntoModel(&state, result)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *DomainResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan DomainModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state DomainModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.ID = state.ID
	plan.DefaultIdpID = state.DefaultIdpID

	current, err := r.client.GetDomain(ctx, plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading domain before update", err.Error())
		return
	}

	updateBody, err := r.buildUpdateBody(plan, current)
	if err != nil {
		resp.Diagnostics.AddError("Invalid domain configuration", err.Error())
		return
	}
	result, err := r.client.UpdateDomain(ctx, plan.ID.ValueString(), updateBody)
	if err != nil {
		resp.Diagnostics.AddError("Error updating domain", err.Error())
		return
	}

	r.readIntoModel(&plan, result)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *DomainResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state DomainModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteDomain(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error deleting domain", err.Error())
	}
}

func (r *DomainResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *DomainResource) buildUpdateBody(plan DomainModel, current map[string]interface{}) (map[string]interface{}, error) {
	body := copyPatchDomainFields(current)

	if current == nil {
		body = map[string]interface{}{
			"dataPlaneId": plan.DataPlaneID.ValueString(),
		}
	}

	settings, settingsProvided, err := domainSettingsJSONToAPI(plan.SettingsJSON)
	if err != nil {
		return nil, err
	}
	if settingsProvided {
		mergeStringInterfaceMap(body, settings)
	}

	body["name"] = plan.Name.ValueString()
	body["enabled"] = plan.Enabled.ValueBool()

	if plan.DataPlaneID.ValueString() != "" {
		body["dataPlaneId"] = plan.DataPlaneID.ValueString()
	}

	if !plan.Description.IsNull() {
		body["description"] = plan.Description.ValueString()
	}

	// OIDC settings
	oidcSettings := nestedMap(body["oidc"])
	clientRegistrationSettings := nestedMap(oidcSettings["clientRegistrationSettings"])
	if current == nil {
		clientRegistrationSettings = map[string]interface{}{
			"allowLocalhostRedirectUri":          false,
			"allowHttpSchemeRedirectUri":         false,
			"allowWildCardRedirectUri":           false,
			"isDynamicClientRegistrationEnabled": false,
		}
	}

	if plan.OIDC != nil {
		clientRegistrationSettings["allowLocalhostRedirectUri"] = plan.OIDC.AllowLocalhostRedirectURI.ValueBool()
		clientRegistrationSettings["allowHttpSchemeRedirectUri"] = plan.OIDC.AllowHTTPSchemeRedirectURI.ValueBool()
		clientRegistrationSettings["allowWildCardRedirectUri"] = plan.OIDC.AllowWildcardRedirectURI.ValueBool()
		clientRegistrationSettings["isDynamicClientRegistrationEnabled"] = plan.OIDC.DynamicClientRegistrationEnabled.ValueBool()
	}
	oidcSettings["clientRegistrationSettings"] = clientRegistrationSettings
	body["oidc"] = oidcSettings

	// Login settings
	loginSettings := nestedMap(body["loginSettings"])
	if current == nil {
		loginSettings = map[string]interface{}{
			"registerEnabled":        false,
			"forgotPasswordEnabled":  false,
			"identifierFirstEnabled": false,
		}
	}
	if plan.LoginSettings != nil {
		loginSettings["registerEnabled"] = plan.LoginSettings.RegisterEnabled.ValueBool()
		loginSettings["forgotPasswordEnabled"] = plan.LoginSettings.ForgotPasswordEnabled.ValueBool()
		loginSettings["identifierFirstEnabled"] = plan.LoginSettings.IdentifierFirstEnabled.ValueBool()
	}
	body["loginSettings"] = loginSettings

	return body, nil
}

func copyPatchDomainFields(current map[string]interface{}) map[string]interface{} {
	allowed := []string{
		"accountSettings",
		"alertEnabled",
		"certificateSettings",
		"corsSettings",
		"dataPlaneId",
		"description",
		"enabled",
		"loginSettings",
		"master",
		"name",
		"oidc",
		"passwordSettings",
		"path",
		"requiredPermissions",
		"saml",
		"scim",
		"secretSettings",
		"secretExpirationSettings",
		"selfServiceAccountManagementSettings",
		"tags",
		"tokenExchangeSettings",
		"uma",
		"vhostMode",
		"vhosts",
		"webAuthnSettings",
	}
	body := make(map[string]interface{}, len(allowed))
	for _, field := range allowed {
		if value, ok := current[field]; ok {
			body[field] = value
		}
	}
	return body
}

func nestedMap(value interface{}) map[string]interface{} {
	if existing, ok := value.(map[string]interface{}); ok {
		copy := make(map[string]interface{}, len(existing))
		for key, child := range existing {
			copy[key] = child
		}
		return copy
	}
	return map[string]interface{}{}
}

func domainSettingsJSONToAPI(value types.String) (map[string]interface{}, bool, error) {
	if value.IsNull() || value.IsUnknown() {
		return nil, false, nil
	}

	var settings map[string]interface{}
	if err := json.Unmarshal([]byte(value.ValueString()), &settings); err != nil {
		return nil, false, fmt.Errorf("settings_json must be a valid JSON object: %w", err)
	}
	if settings == nil {
		return nil, false, fmt.Errorf("settings_json must be a valid JSON object")
	}
	return settings, true, nil
}

func mergeStringInterfaceMap(dst, src map[string]interface{}) {
	for key, value := range src {
		srcMap, srcIsMap := value.(map[string]interface{})
		dstMap, dstIsMap := dst[key].(map[string]interface{})
		if srcIsMap && dstIsMap {
			mergeStringInterfaceMap(dstMap, srcMap)
			continue
		}
		dst[key] = value
	}
}

func (r *DomainResource) readIntoModel(model *DomainModel, data map[string]interface{}) {
	if id, ok := data["id"].(string); ok {
		model.ID = types.StringValue(id)
		model.DefaultIdpID = types.StringValue("default-idp-" + id)
	}
	if name, ok := data["name"].(string); ok {
		model.Name = types.StringValue(name)
	}
	if desc, ok := data["description"].(string); ok {
		model.Description = types.StringValue(desc)
	} else {
		model.Description = types.StringNull()
	}
	if enabled, ok := data["enabled"].(bool); ok {
		model.Enabled = types.BoolValue(enabled)
	}
	if dataPlaneID, ok := data["dataPlaneId"].(string); ok && dataPlaneID != "" {
		model.DataPlaneID = types.StringValue(dataPlaneID)
	} else if model.DataPlaneID.IsNull() || model.DataPlaneID.IsUnknown() {
		model.DataPlaneID = types.StringValue("default")
	}

	// Read OIDC settings
	if oidcData, ok := data["oidc"].(map[string]interface{}); ok {
		if crs, ok := oidcData["clientRegistrationSettings"].(map[string]interface{}); ok {
			if model.OIDC == nil {
				model.OIDC = &OIDCModel{}
			}
			model.OIDC.AllowLocalhostRedirectURI = types.BoolValue(getBool(crs, "allowLocalhostRedirectUri"))
			model.OIDC.AllowHTTPSchemeRedirectURI = types.BoolValue(getBool(crs, "allowHttpSchemeRedirectUri"))
			model.OIDC.AllowWildcardRedirectURI = types.BoolValue(getBool(crs, "allowWildCardRedirectUri"))
			model.OIDC.DynamicClientRegistrationEnabled = types.BoolValue(getBool(crs, "isDynamicClientRegistrationEnabled"))
		}
	}

	// Read login settings
	if ls, ok := data["loginSettings"].(map[string]interface{}); ok {
		if model.LoginSettings == nil {
			model.LoginSettings = &LoginSettingsModel{}
		}
		model.LoginSettings.RegisterEnabled = types.BoolValue(getBool(ls, "registerEnabled"))
		model.LoginSettings.ForgotPasswordEnabled = types.BoolValue(getBool(ls, "forgotPasswordEnabled"))
		model.LoginSettings.IdentifierFirstEnabled = types.BoolValue(getBool(ls, "identifierFirstEnabled"))
	}
}

func getBool(m map[string]interface{}, key string) bool {
	if v, ok := m[key].(bool); ok {
		return v
	}
	return false
}
