package protectedresourcesecret

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
	_ resource.Resource                = &ProtectedResourceSecretResource{}
	_ resource.ResourceWithImportState = &ProtectedResourceSecretResource{}
)

type ProtectedResourceSecretResource struct {
	client *client.Client
}

func NewProtectedResourceSecretResource() resource.Resource {
	return &ProtectedResourceSecretResource{}
}

func (r *ProtectedResourceSecretResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_protected_resource_secret"
}

func (r *ProtectedResourceSecretResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Gravitee AM protected resource client secret",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the protected resource secret",
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
			"protected_resource_id": schema.StringAttribute{
				Required:    true,
				Description: "The ID of the protected resource",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the protected resource secret",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"renew_trigger": schema.StringAttribute{
				Optional:    true,
				Description: "Arbitrary value used to explicitly renew the secret. Changing this value calls the Gravitee AM renewal endpoint and stores the newly returned secret.",
			},
			"secret": schema.StringAttribute{
				Computed:    true,
				Sensitive:   true,
				Description: "The generated client secret value. Gravitee AM only returns the clear value at creation or renewal time.",
			},
			"settings_id": schema.StringAttribute{
				Computed:    true,
				Description: "The secret settings ID returned by Gravitee AM",
			},
			"expires_at": schema.StringAttribute{
				Computed:    true,
				Description: "The expiration timestamp of the secret when configured by Gravitee AM",
			},
		},
	}
}

func (r *ProtectedResourceSecretResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ProtectedResourceSecretResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ProtectedResourceSecretModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.CreateProtectedResourceSecret(ctx, plan.DomainID.ValueString(), plan.ProtectedResourceID.ValueString(), buildBody(plan))
	if err != nil {
		resp.Diagnostics.AddError("Error creating protected resource secret", err.Error())
		return
	}

	readIntoModel(&plan, result, plan.Secret)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ProtectedResourceSecretResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ProtectedResourceSecretModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	secret, err := r.findSecret(ctx, state.DomainID.ValueString(), state.ProtectedResourceID.ValueString(), state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading protected resource secret", err.Error())
		return
	}

	readIntoModel(&state, secret, state.Secret)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ProtectedResourceSecretResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ProtectedResourceSecretModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state ProtectedResourceSecretModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.ID = state.ID
	plan.Secret = state.Secret
	plan.SettingsID = state.SettingsID
	plan.ExpiresAt = state.ExpiresAt

	if shouldRenew(plan.RenewTrigger, state.RenewTrigger) {
		result, err := r.client.RenewProtectedResourceSecret(ctx, plan.DomainID.ValueString(), plan.ProtectedResourceID.ValueString(), plan.ID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Error renewing protected resource secret", err.Error())
			return
		}
		readIntoModel(&plan, result, state.Secret)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ProtectedResourceSecretResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ProtectedResourceSecretModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteProtectedResourceSecret(ctx, state.DomainID.ValueString(), state.ProtectedResourceID.ValueString(), state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Error deleting protected resource secret", err.Error())
	}
}

func (r *ProtectedResourceSecretResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import format: domain_id/protected_resource_id/secret_id
	domainID, protectedResourceID, secretID, ok := parseImportID(req.ID)
	if !ok {
		resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("Expected format: domain_id/protected_resource_id/secret_id, got: %s", req.ID))
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("domain_id"), domainID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("protected_resource_id"), protectedResourceID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), secretID)...)
}

func parseImportID(id string) (string, string, string, bool) {
	parts := strings.Split(id, "/")
	if len(parts) != 3 {
		return "", "", "", false
	}
	return parts[0], parts[1], parts[2], true
}

func buildBody(plan ProtectedResourceSecretModel) map[string]interface{} {
	return map[string]interface{}{
		"name": plan.Name.ValueString(),
	}
}

func (r *ProtectedResourceSecretResource) findSecret(ctx context.Context, domainID, protectedResourceID, secretID string) (map[string]interface{}, error) {
	secrets, err := r.client.ListProtectedResourceSecrets(ctx, domainID, protectedResourceID)
	if err != nil {
		return nil, err
	}
	for _, secret := range secrets {
		if id, ok := secret["id"].(string); ok && id == secretID {
			return secret, nil
		}
	}
	return nil, fmt.Errorf("protected resource secret not found: %w", client.ErrNotFound)
}

func readIntoModel(model *ProtectedResourceSecretModel, data map[string]interface{}, preservedSecret types.String) {
	if id, ok := data["id"].(string); ok {
		model.ID = types.StringValue(id)
	}
	if name, ok := data["name"].(string); ok {
		model.Name = types.StringValue(name)
	}
	if secret, ok := data["secret"].(string); ok && secret != "" {
		model.Secret = types.StringValue(secret)
	} else if !preservedSecret.IsUnknown() {
		model.Secret = preservedSecret
	} else {
		model.Secret = types.StringNull()
	}
	if settingsID, ok := data["settingsId"].(string); ok {
		model.SettingsID = types.StringValue(settingsID)
	} else {
		model.SettingsID = types.StringNull()
	}
	if expiresAt, ok := data["expiresAt"].(string); ok {
		model.ExpiresAt = types.StringValue(expiresAt)
	} else {
		model.ExpiresAt = types.StringNull()
	}
}

func shouldRenew(planTrigger, stateTrigger types.String) bool {
	if planTrigger.IsNull() || planTrigger.IsUnknown() {
		return false
	}
	if stateTrigger.IsNull() || stateTrigger.IsUnknown() {
		return true
	}
	return planTrigger.ValueString() != stateTrigger.ValueString()
}
