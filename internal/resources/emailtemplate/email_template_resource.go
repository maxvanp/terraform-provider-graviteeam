package emailtemplate

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
	_ resource.Resource                = &EmailTemplateResource{}
	_ resource.ResourceWithImportState = &EmailTemplateResource{}
)

var validEmailTemplates = map[string]bool{
	"REGISTRATION_CONFIRMATION": true,
	"RESET_PASSWORD":            true,
	"BLOCKED_ACCOUNT":           true,
	"MFA_CHALLENGE":             true,
	"CERTIFICATE_EXPIRATION":    true,
	"REGISTRATION_VERIFY":       true,
}

type EmailTemplateResource struct {
	client *client.Client
}

func NewEmailTemplateResource() resource.Resource {
	return &EmailTemplateResource{}
}

func (r *EmailTemplateResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_email_template"
}

func (r *EmailTemplateResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Gravitee AM Email Template (custom email content for registration confirmation, password reset, etc.)",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the email template",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"domain_id": schema.StringAttribute{
				Required:    true,
				Description: "The ID of the domain this email template belongs to",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"application_id": schema.StringAttribute{
				Optional:    true,
				Description: "The ID of the application (for app-level override). If not set, the email template applies at domain level.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"template": schema.StringAttribute{
				Required:    true,
				Description: "The email template type (e.g. REGISTRATION_CONFIRMATION, RESET_PASSWORD, BLOCKED_ACCOUNT, MFA_CHALLENGE)",
				Validators:  []validator.String{emailTemplateValidator{}},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"enabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				Description: "Whether the custom email template is enabled",
			},
			"from": schema.StringAttribute{
				Required:    true,
				Description: "Sender email address",
			},
			"from_name": schema.StringAttribute{
				Optional:    true,
				Description: "Sender display name",
			},
			"subject": schema.StringAttribute{
				Required:    true,
				Description: "Email subject line",
			},
			"content": schema.StringAttribute{
				Required:    true,
				Description: "The HTML content of the email (supports FreeMarker expressions)",
			},
			"expires_after": schema.Int64Attribute{
				Required:    true,
				Description: "Token/link validity duration in seconds (e.g. 86400 for 24 hours)",
			},
		},
	}
}

func (r *EmailTemplateResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *EmailTemplateResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan EmailTemplateModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := r.buildBody(plan, nil)
	body["template"] = plan.Template.ValueString()

	appID := ""
	if !plan.ApplicationID.IsNull() && !plan.ApplicationID.IsUnknown() {
		appID = plan.ApplicationID.ValueString()
	}

	result, err := r.client.CreateEmail(ctx, plan.DomainID.ValueString(), appID, body)
	if err != nil {
		resp.Diagnostics.AddError("Error creating email template", err.Error())
		return
	}

	plan.ID = types.StringValue(result["id"].(string))

	r.readIntoModel(&plan, result)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *EmailTemplateResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state EmailTemplateModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	appID := ""
	if !state.ApplicationID.IsNull() && !state.ApplicationID.IsUnknown() {
		appID = state.ApplicationID.ValueString()
	}

	result, err := r.client.GetEmail(ctx, state.DomainID.ValueString(), appID, state.Template.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading email template", err.Error())
		return
	}

	r.readIntoModel(&state, result)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *EmailTemplateResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan EmailTemplateModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state EmailTemplateModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.ID = state.ID

	body := r.buildBody(plan, &state)

	appID := ""
	if !plan.ApplicationID.IsNull() && !plan.ApplicationID.IsUnknown() {
		appID = plan.ApplicationID.ValueString()
	}

	result, err := r.client.UpdateEmail(ctx, plan.DomainID.ValueString(), appID, plan.ID.ValueString(), body)
	if err != nil {
		resp.Diagnostics.AddError("Error updating email template", err.Error())
		return
	}

	r.readIntoModel(&plan, result)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *EmailTemplateResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state EmailTemplateModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	appID := ""
	if !state.ApplicationID.IsNull() && !state.ApplicationID.IsUnknown() {
		appID = state.ApplicationID.ValueString()
	}

	err := r.client.DeleteEmail(ctx, state.DomainID.ValueString(), appID, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Error deleting email template", err.Error())
	}
}

func (r *EmailTemplateResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import format: domain_id/template or domain_id/app_id/template
	// The API retrieves email templates by template type (not by ID),
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

func (r *EmailTemplateResource) buildBody(plan EmailTemplateModel, state *EmailTemplateModel) map[string]interface{} {
	body := map[string]interface{}{
		"enabled":      plan.Enabled.ValueBool(),
		"from":         plan.From.ValueString(),
		"subject":      plan.Subject.ValueString(),
		"content":      plan.Content.ValueString(),
		"expiresAfter": plan.ExpiresAfter.ValueInt64(),
	}

	if !plan.FromName.IsNull() && !plan.FromName.IsUnknown() {
		body["fromName"] = plan.FromName.ValueString()
	} else if state != nil && !state.FromName.IsNull() && !state.FromName.IsUnknown() {
		body["fromName"] = ""
	}

	return body
}

func (r *EmailTemplateResource) readIntoModel(model *EmailTemplateModel, data map[string]interface{}) {
	if id, ok := data["id"].(string); ok {
		model.ID = types.StringValue(id)
	}
	if v, ok := data["template"].(string); ok {
		model.Template = types.StringValue(strings.ToUpper(v))
	}
	if v, ok := data["enabled"].(bool); ok {
		model.Enabled = types.BoolValue(v)
	}
	if v, ok := data["from"].(string); ok {
		model.From = types.StringValue(v)
	}
	if v, ok := data["fromName"].(string); ok && v != "" {
		model.FromName = types.StringValue(v)
	} else if model.FromName.IsNull() {
		model.FromName = types.StringNull()
	}
	if v, ok := data["subject"].(string); ok {
		model.Subject = types.StringValue(v)
	}
	if v, ok := data["content"].(string); ok {
		model.Content = types.StringValue(v)
	}
	if v, ok := data["expiresAfter"]; ok {
		switch n := v.(type) {
		case float64:
			model.ExpiresAfter = types.Int64Value(int64(n))
		case int64:
			model.ExpiresAfter = types.Int64Value(n)
		}
	}
}

// Validator for email template type
type emailTemplateValidator struct{}

func (v emailTemplateValidator) Description(_ context.Context) string {
	return "template must be a valid Gravitee AM email template type"
}

func (v emailTemplateValidator) MarkdownDescription(_ context.Context) string {
	return "template must be a valid Gravitee AM email template type"
}

func (v emailTemplateValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	value := req.ConfigValue.ValueString()
	if !validEmailTemplates[value] {
		resp.Diagnostics.AddError(
			"Invalid template",
			fmt.Sprintf("email template must be one of: REGISTRATION_CONFIRMATION, RESET_PASSWORD, BLOCKED_ACCOUNT, MFA_CHALLENGE, CERTIFICATE_EXPIRATION, REGISTRATION_VERIFY. Got: %s", value),
		)
	}
}
