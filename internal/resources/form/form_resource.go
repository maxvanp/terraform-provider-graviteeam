package form

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
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/maxvanp/terraform-provider-graviteeam/internal/client"
)

var (
	_ resource.Resource                = &FormResource{}
	_ resource.ResourceWithImportState = &FormResource{}
)

var validTemplates = map[string]bool{
	"LOGIN":                      true,
	"REGISTRATION":               true,
	"REGISTRATION_CONFIRMATION":  true,
	"FORGOT_PASSWORD":            true,
	"RESET_PASSWORD":             true,
	"OAUTH2_USER_CONSENT":        true,
	"MFA_ENROLL":                 true,
	"MFA_CHALLENGE":              true,
	"MFA_CHALLENGE_ALTERNATIVES": true,
	"MFA_RECOVERY_CODE":          true,
	"BLOCKED_ACCOUNT":            true,
	"COMPLETE_PROFILE":           true,
	"WEBAUTHN_REGISTER":          true,
	"WEBAUTHN_REGISTER_SUCCESS":  true,
	"WEBAUTHN_LOGIN":             true,
	"IDENTIFIER_FIRST_LOGIN":     true,
	"ERROR":                      true,
	"CERTIFICATE_EXPIRATION":     true,
	"REGISTRATION_VERIFY":        true,
}

type FormResource struct {
	client *client.Client
}

func NewFormResource() resource.Resource {
	return &FormResource{}
}

func (r *FormResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_form"
}

func (r *FormResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Gravitee AM Form (custom login, registration, and other page templates)",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the form",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"domain_id": schema.StringAttribute{
				Required:    true,
				Description: "The ID of the domain this form belongs to",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"application_id": schema.StringAttribute{
				Optional:    true,
				Description: "The ID of the application (for app-level override). If not set, the form applies at domain level.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"template": schema.StringAttribute{
				Required:    true,
				Description: "The template type (e.g. LOGIN, REGISTRATION, FORGOT_PASSWORD, etc.)",
				Validators:  []validator.String{templateValidator{}},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"enabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				Description: "Whether the custom form is enabled",
			},
			"content": schema.StringAttribute{
				Required:    true,
				Description: "The HTML content of the form (supports Thymeleaf expressions)",
			},
		},
	}
}

func (r *FormResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *FormResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan FormModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]interface{}{
		"template": plan.Template.ValueString(),
		"enabled":  plan.Enabled.ValueBool(),
		"content":  plan.Content.ValueString(),
	}

	appID := ""
	if !plan.ApplicationID.IsNull() && !plan.ApplicationID.IsUnknown() {
		appID = plan.ApplicationID.ValueString()
	}

	result, err := r.client.CreateForm(ctx, plan.DomainID.ValueString(), appID, body)
	if err != nil {
		resp.Diagnostics.AddError("Error creating form", err.Error())
		return
	}

	plan.ID = types.StringValue(result["id"].(string))

	r.readIntoModel(&plan, result)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *FormResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state FormModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	appID := ""
	if !state.ApplicationID.IsNull() && !state.ApplicationID.IsUnknown() {
		appID = state.ApplicationID.ValueString()
	}

	result, err := r.client.GetForm(ctx, state.DomainID.ValueString(), appID, state.Template.ValueString())
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading form", err.Error())
		return
	}

	r.readIntoModel(&state, result)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *FormResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan FormModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state FormModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.ID = state.ID

	body := map[string]interface{}{
		"enabled": plan.Enabled.ValueBool(),
		"content": plan.Content.ValueString(),
	}

	appID := ""
	if !plan.ApplicationID.IsNull() && !plan.ApplicationID.IsUnknown() {
		appID = plan.ApplicationID.ValueString()
	}

	result, err := r.client.UpdateForm(ctx, plan.DomainID.ValueString(), appID, plan.ID.ValueString(), body)
	if err != nil {
		resp.Diagnostics.AddError("Error updating form", err.Error())
		return
	}

	r.readIntoModel(&plan, result)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *FormResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state FormModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	appID := ""
	if !state.ApplicationID.IsNull() && !state.ApplicationID.IsUnknown() {
		appID = state.ApplicationID.ValueString()
	}

	err := r.client.DeleteForm(ctx, state.DomainID.ValueString(), appID, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error deleting form", err.Error())
	}
}

func (r *FormResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import format: domain_id/template or domain_id/app_id/template
	// The API retrieves forms by template type (not by ID),
	// so the import ID must include the template type.
	parts := strings.Split(req.ID, "/")
	switch len(parts) {
	case 2:
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("domain_id"), parts[0])...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("template"), parts[1])...)
	case 3:
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("domain_id"), parts[0])...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("application_id"), parts[1])...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("template"), parts[2])...)
	default:
		resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("Expected format: domain_id/template or domain_id/app_id/template, got: %s", req.ID))
	}
}

func (r *FormResource) readIntoModel(model *FormModel, data map[string]interface{}) {
	if id, ok := data["id"].(string); ok {
		model.ID = types.StringValue(id)
	}
	if v, ok := data["template"].(string); ok {
		model.Template = types.StringValue(strings.ToUpper(v))
	}
	if v, ok := data["enabled"].(bool); ok {
		model.Enabled = types.BoolValue(v)
	}
	if v, ok := data["content"].(string); ok {
		model.Content = types.StringValue(v)
	}
}

// Validator for template type
type templateValidator struct{}

func (v templateValidator) Description(_ context.Context) string {
	return "template must be a valid Gravitee AM form template type"
}

func (v templateValidator) MarkdownDescription(_ context.Context) string {
	return "template must be a valid Gravitee AM form template type"
}

func (v templateValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	value := req.ConfigValue.ValueString()
	if !validTemplates[value] {
		resp.Diagnostics.AddError(
			"Invalid template",
			fmt.Sprintf("template must be one of: LOGIN, REGISTRATION, REGISTRATION_CONFIRMATION, FORGOT_PASSWORD, RESET_PASSWORD, OAUTH2_USER_CONSENT, MFA_ENROLL, MFA_CHALLENGE, MFA_CHALLENGE_ALTERNATIVES, MFA_RECOVERY_CODE, BLOCKED_ACCOUNT, COMPLETE_PROFILE, WEBAUTHN_REGISTER, WEBAUTHN_REGISTER_SUCCESS, WEBAUTHN_LOGIN, IDENTIFIER_FIRST_LOGIN, ERROR, CERTIFICATE_EXPIRATION, REGISTRATION_VERIFY. Got: %s", value),
		)
	}
}
