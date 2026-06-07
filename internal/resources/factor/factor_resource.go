package factor

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/maxvanp/terraform-provider-graviteeam/internal/client"
)

var (
	_ resource.Resource                = &FactorResource{}
	_ resource.ResourceWithImportState = &FactorResource{}
)

// Map from user-friendly factor_type to Gravitee AM plugin type
var factorTypeToPluginType = map[string]string{
	"TOTP":  "otp-am-factor",
	"EMAIL": "email-am-factor",
	"SMS":   "sms-am-factor",
}

var pluginTypeToFactorType = map[string]string{
	"otp-am-factor":   "TOTP",
	"email-am-factor": "EMAIL",
	"sms-am-factor":   "SMS",
}

// API returns lowercase factorType values in GET responses
var apiFactorTypeToUserType = map[string]string{
	"otp":   "TOTP",
	"email": "EMAIL",
	"sms":   "SMS",
	"TOTP":  "TOTP",
	"EMAIL": "EMAIL",
	"SMS":   "SMS",
}

type FactorResource struct {
	client *client.Client
}

func NewFactorResource() resource.Resource {
	return &FactorResource{}
}

func (r *FactorResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_factor"
}

func (r *FactorResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Gravitee AM MFA Factor",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the factor",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"domain_id": schema.StringAttribute{
				Required:    true,
				Description: "The ID of the domain this factor belongs to",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the factor",
			},
			"factor_type": schema.StringAttribute{
				Required:    true,
				Description: "The type of factor: TOTP, EMAIL, or SMS",
				Validators:  []validator.String{factorTypeValidator{}},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *FactorResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *FactorResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan FactorModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	factorType := plan.FactorType.ValueString()
	pluginType, ok := factorTypeToPluginType[factorType]
	if !ok {
		resp.Diagnostics.AddError("Invalid factor_type", fmt.Sprintf("Unknown factor type: %s", factorType))
		return
	}

	body := buildCreateBody(plan, pluginType)

	result, err := r.client.CreateFactor(ctx, plan.DomainID.ValueString(), body)
	if err != nil {
		resp.Diagnostics.AddError("Error creating factor", err.Error())
		return
	}

	plan.ID = types.StringValue(result["id"].(string))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *FactorResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state FactorModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.GetFactor(ctx, state.DomainID.ValueString(), state.ID.ValueString())
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading factor", err.Error())
		return
	}

	readIntoModel(&state, result)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *FactorResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan FactorModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state FactorModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.ID = state.ID

	factorType := plan.FactorType.ValueString()
	pluginType, ok := factorTypeToPluginType[factorType]
	if !ok {
		resp.Diagnostics.AddError("Invalid factor_type", fmt.Sprintf("Unknown factor type: %s", factorType))
		return
	}

	body := buildUpdateBody(plan, pluginType)

	_, err := r.client.UpdateFactor(ctx, plan.DomainID.ValueString(), plan.ID.ValueString(), body)
	if err != nil {
		resp.Diagnostics.AddError("Error updating factor", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *FactorResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state FactorModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteFactor(ctx, state.DomainID.ValueString(), state.ID.ValueString())
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			return
		}
		resp.Diagnostics.AddError("Error deleting factor", err.Error())
	}
}

func (r *FactorResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import format: domain_id/factor_id
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 {
		resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("Expected format: domain_id/factor_id, got: %s", req.ID))
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("domain_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
}

func buildCreateBody(plan FactorModel, pluginType string) map[string]interface{} {
	return map[string]interface{}{
		"name":          plan.Name.ValueString(),
		"type":          pluginType,
		"factorType":    plan.FactorType.ValueString(),
		"configuration": "{}",
	}
}

func buildUpdateBody(plan FactorModel, pluginType string) map[string]interface{} {
	return map[string]interface{}{
		"name":          plan.Name.ValueString(),
		"type":          pluginType,
		"configuration": "{}",
	}
}

func readIntoModel(model *FactorModel, result map[string]interface{}) {
	if name, ok := result["name"].(string); ok {
		model.Name = types.StringValue(name)
	}
	if ft, ok := result["factorType"].(string); ok {
		if mapped, ok := apiFactorTypeToUserType[ft]; ok {
			model.FactorType = types.StringValue(mapped)
		} else {
			model.FactorType = types.StringValue(ft)
		}
	} else if pluginT, ok := result["type"].(string); ok {
		if ft, ok := pluginTypeToFactorType[pluginT]; ok {
			model.FactorType = types.StringValue(ft)
		}
	}
}

// Validator for factor_type
type factorTypeValidator struct{}

func (v factorTypeValidator) Description(_ context.Context) string {
	return "factor_type must be one of: TOTP, EMAIL, SMS"
}

func (v factorTypeValidator) MarkdownDescription(_ context.Context) string {
	return "factor_type must be one of: `TOTP`, `EMAIL`, `SMS`"
}

func (v factorTypeValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	value := req.ConfigValue.ValueString()
	if _, ok := factorTypeToPluginType[value]; !ok {
		resp.Diagnostics.AddError(
			"Invalid factor_type",
			fmt.Sprintf("factor_type must be one of: TOTP, EMAIL, SMS. Got: %s", value),
		)
	}
}
