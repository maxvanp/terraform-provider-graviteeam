package alerttrigger

import (
	"context"
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
	"github.com/maxvanp/terraform-provider-graviteeam/internal/resources/importid"
)

var (
	_ resource.Resource                = &AlertTriggerResource{}
	_ resource.ResourceWithImportState = &AlertTriggerResource{}
)

type AlertTriggerResource struct {
	client *client.Client
}

func NewAlertTriggerResource() resource.Resource {
	return &AlertTriggerResource{}
}

func (r *AlertTriggerResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_alert_trigger"
}

func (r *AlertTriggerResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Gravitee AM alert trigger",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The Terraform ID of the alert trigger",
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
			"type": schema.StringAttribute{
				Required:    true,
				Description: "The alert trigger type: TOO_MANY_LOGIN_FAILURES or RISK_ASSESSMENT",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"enabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				Description: "Whether the alert trigger is enabled",
			},
			"alert_notifier_ids": schema.SetAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Description: "Set of alert notifier IDs used by this trigger",
			},
		},
	}
}

func (r *AlertTriggerResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *AlertTriggerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan AlertTriggerModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := validateTriggerType(plan.Type.ValueString()); err != nil {
		resp.Diagnostics.AddError("Invalid alert trigger type", err.Error())
		return
	}

	result, err := r.patch(ctx, &plan, plan.Enabled.ValueBool())
	if err != nil {
		resp.Diagnostics.AddError("Error creating alert trigger", err.Error())
		return
	}

	readIntoModel(&plan, result)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *AlertTriggerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state AlertTriggerModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	triggers, err := r.client.ListAlertTriggers(ctx, state.DomainID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading alert trigger", err.Error())
		return
	}

	trigger, ok := findTrigger(triggers, state.Type.ValueString())
	if !ok {
		resp.State.RemoveResource(ctx)
		return
	}

	readIntoModel(&state, trigger)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *AlertTriggerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan AlertTriggerModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := validateTriggerType(plan.Type.ValueString()); err != nil {
		resp.Diagnostics.AddError("Invalid alert trigger type", err.Error())
		return
	}

	result, err := r.patch(ctx, &plan, plan.Enabled.ValueBool())
	if err != nil {
		resp.Diagnostics.AddError("Error updating alert trigger", err.Error())
		return
	}

	readIntoModel(&plan, result)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *AlertTriggerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state AlertTriggerModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	state.AlertNotifierIDs = types.SetNull(types.StringType)
	_, err := r.patch(ctx, &state, false)
	if err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Error deleting alert trigger", err.Error())
	}
}

func (r *AlertTriggerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 || !importid.Valid(parts) {
		resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("Expected format: domain_id/type, got: %s", req.ID))
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("domain_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("type"), normalizeTriggerType(parts[1]))...)
}

func (r *AlertTriggerResource) patch(ctx context.Context, model *AlertTriggerModel, enabled bool) (map[string]interface{}, error) {
	body := []map[string]interface{}{
		{
			"type":           normalizeTriggerType(model.Type.ValueString()),
			"enabled":        enabled,
			"alertNotifiers": stringValues(model.AlertNotifierIDs),
		},
	}
	triggers, err := r.client.PatchAlertTriggers(ctx, model.DomainID.ValueString(), body)
	if err != nil {
		return nil, err
	}
	trigger, ok := findTrigger(triggers, model.Type.ValueString())
	if !ok {
		return nil, fmt.Errorf("alert trigger %s not found in API response", model.Type.ValueString())
	}
	return trigger, nil
}

func findTrigger(triggers []map[string]interface{}, triggerType string) (map[string]interface{}, bool) {
	normalized := normalizeTriggerType(triggerType)
	for _, trigger := range triggers {
		if t, ok := trigger["type"].(string); ok && normalizeTriggerType(t) == normalized {
			return trigger, true
		}
	}
	return nil, false
}

func readIntoModel(model *AlertTriggerModel, trigger map[string]interface{}) {
	model.ID = types.StringValue(model.DomainID.ValueString() + "/" + normalizeTriggerType(model.Type.ValueString()))
	model.Type = types.StringValue(normalizeTriggerType(model.Type.ValueString()))
	if t, ok := trigger["type"].(string); ok && t != "" {
		model.Type = types.StringValue(normalizeTriggerType(t))
		model.ID = types.StringValue(model.DomainID.ValueString() + "/" + normalizeTriggerType(t))
	}
	if enabled, ok := trigger["enabled"].(bool); ok {
		model.Enabled = types.BoolValue(enabled)
	}
	if values, ok := trigger["alertNotifiers"].([]interface{}); ok {
		elements := make([]attr.Value, 0, len(values))
		for _, value := range values {
			if s, ok := value.(string); ok {
				elements = append(elements, types.StringValue(s))
			}
		}
		setValue, _ := types.SetValue(types.StringType, elements)
		model.AlertNotifierIDs = setValue
	} else {
		setValue, _ := types.SetValue(types.StringType, nil)
		model.AlertNotifierIDs = setValue
	}
}

func stringValues(values types.Set) []string {
	if values.IsNull() || values.IsUnknown() {
		return []string{}
	}
	result := make([]string, 0, len(values.Elements()))
	for _, value := range values.Elements() {
		if stringValue, ok := value.(types.String); ok && !stringValue.IsNull() && !stringValue.IsUnknown() {
			result = append(result, stringValue.ValueString())
		}
	}
	return result
}

func validateTriggerType(value string) error {
	switch normalizeTriggerType(value) {
	case "TOO_MANY_LOGIN_FAILURES", "RISK_ASSESSMENT":
		return nil
	default:
		return fmt.Errorf("expected TOO_MANY_LOGIN_FAILURES or RISK_ASSESSMENT, got %s", value)
	}
}

func normalizeTriggerType(value string) string {
	return strings.ToUpper(value)
}
