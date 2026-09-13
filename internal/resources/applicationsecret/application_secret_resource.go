package applicationsecret

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
	"github.com/maxvanp/terraform-provider-graviteeam/internal/resources/importid"
)

var (
	_ resource.Resource                = &ApplicationSecretResource{}
	_ resource.ResourceWithImportState = &ApplicationSecretResource{}
)

type ApplicationSecretResource struct {
	client *client.Client
}

func NewApplicationSecretResource() resource.Resource {
	return &ApplicationSecretResource{}
}

func (r *ApplicationSecretResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_application_secret"
}

func (r *ApplicationSecretResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Gravitee AM application client secret",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the application secret",
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
			"application_id": schema.StringAttribute{
				Required:    true,
				Description: "The ID of the application",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the application secret",
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

func (r *ApplicationSecretResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ApplicationSecretResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ApplicationSecretModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.CreateApplicationSecret(ctx, plan.DomainID.ValueString(), plan.ApplicationID.ValueString(), map[string]interface{}{
		"name": plan.Name.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error creating application secret", err.Error())
		return
	}

	readIntoModel(&plan, result, plan.Secret)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ApplicationSecretResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ApplicationSecretModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	secret, err := r.findSecret(ctx, state.DomainID.ValueString(), state.ApplicationID.ValueString(), state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading application secret", err.Error())
		return
	}

	readIntoModel(&state, secret, state.Secret)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ApplicationSecretResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ApplicationSecretModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state ApplicationSecretModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.ID = state.ID
	plan.Secret = state.Secret
	plan.SettingsID = state.SettingsID
	plan.ExpiresAt = state.ExpiresAt

	if shouldRenew(plan.RenewTrigger, state.RenewTrigger) {
		result, err := r.client.RenewApplicationSecret(ctx, plan.DomainID.ValueString(), plan.ApplicationID.ValueString(), plan.ID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Error renewing application secret", err.Error())
			return
		}
		readIntoModel(&plan, result, state.Secret)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ApplicationSecretResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ApplicationSecretModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteApplicationSecret(ctx, state.DomainID.ValueString(), state.ApplicationID.ValueString(), state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Error deleting application secret", err.Error())
	}
}

func (r *ApplicationSecretResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import format: domain_id/application_id/secret_id
	parts := strings.Split(req.ID, "/")
	if len(parts) != 3 || !importid.Valid(parts) {
		resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("Expected format: domain_id/application_id/secret_id, got: %s", req.ID))
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("domain_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("application_id"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[2])...)
}

func (r *ApplicationSecretResource) findSecret(ctx context.Context, domainID, applicationID, secretID string) (map[string]interface{}, error) {
	secrets, err := r.client.ListApplicationSecrets(ctx, domainID, applicationID)
	if err != nil {
		return nil, err
	}
	for _, secret := range secrets {
		if id, ok := secret["id"].(string); ok && id == secretID {
			return secret, nil
		}
	}
	return nil, fmt.Errorf("application secret not found: %w", client.ErrNotFound)
}

func readIntoModel(model *ApplicationSecretModel, data map[string]interface{}, preservedSecret types.String) {
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
