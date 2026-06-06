package orgform

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
	_ resource.Resource                = &OrgFormResource{}
	_ resource.ResourceWithImportState = &OrgFormResource{}
)

var validOrgTemplates = map[string]bool{
	"LOGIN":                      true,
	"REGISTRATION":               true,
	"REGISTRATION_CONFIRMATION":  true,
	"REGISTRATION_VERIFY":        true,
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
	"CBA_LOGIN":                  true,
	"MAGIC_LINK_LOGIN":           true,
	"MAGIC_LINK":                 true,
	"IDENTIFIER_FIRST_LOGIN":     true,
	"ERROR":                      true,
	"CERTIFICATE_EXPIRATION":     true,
	"CLIENT_SECRET_EXPIRATION":   true,
	"VERIFY_ATTEMPT":             true,
}

type OrgFormResource struct {
	client *client.Client
}

func NewOrgFormResource() resource.Resource {
	return &OrgFormResource{}
}

func (r *OrgFormResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_org_form"
}

func (r *OrgFormResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Gravitee AM organization form template",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the organization form",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
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

func (r *OrgFormResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *OrgFormResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan OrgFormModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.CreateOrgForm(ctx, map[string]interface{}{
		"template": plan.Template.ValueString(),
		"enabled":  plan.Enabled.ValueBool(),
		"content":  plan.Content.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error creating organization form", err.Error())
		return
	}

	readIntoModel(&plan, result)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *OrgFormResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state OrgFormModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.GetOrgForm(ctx, state.Template.ValueString())
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading organization form", err.Error())
		return
	}

	readIntoModel(&state, result)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *OrgFormResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan OrgFormModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state OrgFormModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.ID = state.ID

	current, err := r.client.GetOrgForm(ctx, state.Template.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading current organization form before update", err.Error())
		return
	}

	body := map[string]interface{}{
		"enabled": plan.Enabled.ValueBool(),
		"content": plan.Content.ValueString(),
	}
	if assets, ok := current["assets"]; ok {
		body["assets"] = assets
	}

	result, err := r.client.UpdateOrgForm(ctx, plan.ID.ValueString(), body)
	if err != nil {
		resp.Diagnostics.AddError("Error updating organization form", err.Error())
		return
	}

	readIntoModel(&plan, result)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *OrgFormResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state OrgFormModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteOrgForm(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error deleting organization form", err.Error())
	}
}

func (r *OrgFormResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("template"), req.ID)...)
}

func readIntoModel(model *OrgFormModel, data map[string]interface{}) {
	if id, ok := data["id"].(string); ok {
		model.ID = types.StringValue(id)
	}
	if template, ok := data["template"].(string); ok {
		model.Template = types.StringValue(strings.ToUpper(template))
	}
	if enabled, ok := data["enabled"].(bool); ok {
		model.Enabled = types.BoolValue(enabled)
	}
	if content, ok := data["content"].(string); ok {
		model.Content = types.StringValue(content)
	}
}

type templateValidator struct{}

func (v templateValidator) Description(_ context.Context) string {
	return "must be a valid organization form template type"
}

func (v templateValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v templateValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	value := strings.ToUpper(req.ConfigValue.ValueString())
	if !validOrgTemplates[value] {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid template type",
			fmt.Sprintf("Template must be one of the supported organization form templates, got: %s", req.ConfigValue.ValueString()),
		)
	}
}
