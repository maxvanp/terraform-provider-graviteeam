package extensiongrant

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
	_ resource.Resource                = &ExtensionGrantResource{}
	_ resource.ResourceWithImportState = &ExtensionGrantResource{}
)

type ExtensionGrantResource struct {
	client *client.Client
}

func NewExtensionGrantResource() resource.Resource {
	return &ExtensionGrantResource{}
}

func (r *ExtensionGrantResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_extension_grant"
}

func (r *ExtensionGrantResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Gravitee AM Extension Grant (e.g. JWT Bearer)",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the extension grant",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"domain_id": schema.StringAttribute{
				Required:    true,
				Description: "The ID of the domain this extension grant belongs to",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the extension grant",
			},
			"type": schema.StringAttribute{
				Required:    true,
				Description: "The plugin type (e.g. jwtbearer-am-extension-grant)",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"grant_type": schema.StringAttribute{
				Required:    true,
				Description: "The OAuth 2.0 grant type URI (e.g. urn:ietf:params:oauth:grant-type:jwt-bearer)",
			},
			"configuration": schema.StringAttribute{
				Required:    true,
				Description: "JSON configuration for the extension grant plugin",
			},
			"identity_provider": schema.StringAttribute{
				Optional:    true,
				Description: "The identity provider ID to use for user resolution",
			},
			"create_user": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Create user if not found",
			},
			"user_exists": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Check if user exists",
			},
		},
	}
}

func (r *ExtensionGrantResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ExtensionGrantResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ExtensionGrantModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]interface{}{
		"name":          plan.Name.ValueString(),
		"type":          plan.Type.ValueString(),
		"grantType":     plan.GrantType.ValueString(),
		"configuration": plan.Configuration.ValueString(),
		"createUser":    plan.CreateUser.ValueBool(),
		"userExists":    plan.UserExists.ValueBool(),
	}

	if !plan.IdentityProvider.IsNull() && !plan.IdentityProvider.IsUnknown() {
		body["identityProvider"] = plan.IdentityProvider.ValueString()
	}

	result, err := r.client.CreateExtensionGrant(ctx, plan.DomainID.ValueString(), body)
	if err != nil {
		resp.Diagnostics.AddError("Error creating extension grant", err.Error())
		return
	}

	plan.ID = types.StringValue(result["id"].(string))

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ExtensionGrantResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ExtensionGrantModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.GetExtensionGrant(ctx, state.DomainID.ValueString(), state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading extension grant", err.Error())
		return
	}

	if name, ok := result["name"].(string); ok {
		state.Name = types.StringValue(name)
	}
	if t, ok := result["type"].(string); ok {
		state.Type = types.StringValue(t)
	}
	if gt, ok := result["grantType"].(string); ok {
		state.GrantType = types.StringValue(gt)
	}
	if cfg, ok := result["configuration"].(string); ok {
		state.Configuration = types.StringValue(cfg)
	}
	if idp, ok := result["identityProvider"].(string); ok && idp != "" {
		state.IdentityProvider = types.StringValue(idp)
	} else if state.IdentityProvider.IsNull() {
		// keep null
	} else {
		state.IdentityProvider = types.StringNull()
	}
	if cu, ok := result["createUser"].(bool); ok {
		state.CreateUser = types.BoolValue(cu)
	}
	if ue, ok := result["userExists"].(bool); ok {
		state.UserExists = types.BoolValue(ue)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ExtensionGrantResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ExtensionGrantModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state ExtensionGrantModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.ID = state.ID

	body := buildUpdateBody(plan, state)

	_, err := r.client.UpdateExtensionGrant(ctx, plan.DomainID.ValueString(), plan.ID.ValueString(), body)
	if err != nil {
		resp.Diagnostics.AddError("Error updating extension grant", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func buildUpdateBody(plan, state ExtensionGrantModel) map[string]interface{} {
	body := map[string]interface{}{
		"name":          plan.Name.ValueString(),
		"type":          plan.Type.ValueString(),
		"grantType":     plan.GrantType.ValueString(),
		"configuration": plan.Configuration.ValueString(),
		"createUser":    plan.CreateUser.ValueBool(),
		"userExists":    plan.UserExists.ValueBool(),
	}

	if !plan.IdentityProvider.IsNull() && !plan.IdentityProvider.IsUnknown() {
		body["identityProvider"] = plan.IdentityProvider.ValueString()
	} else if !state.IdentityProvider.IsNull() {
		body["identityProvider"] = ""
	}

	return body
}

func (r *ExtensionGrantResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ExtensionGrantModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteExtensionGrant(ctx, state.DomainID.ValueString(), state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error deleting extension grant", err.Error())
	}
}

func (r *ExtensionGrantResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import format: domain_id/extension_grant_id
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 {
		resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("Expected format: domain_id/extension_grant_id, got: %s", req.ID))
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("domain_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
}
