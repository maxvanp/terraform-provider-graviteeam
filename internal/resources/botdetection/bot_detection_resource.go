package botdetection

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
	_ resource.Resource                = &BotDetectionResource{}
	_ resource.ResourceWithImportState = &BotDetectionResource{}
)

type BotDetectionResource struct {
	client *client.Client
}

func NewBotDetectionResource() resource.Resource {
	return &BotDetectionResource{}
}

func (r *BotDetectionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_bot_detection"
}

func (r *BotDetectionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Gravitee AM Bot Detection plugin",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the bot detection",
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
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the bot detection",
			},
			"type": schema.StringAttribute{
				Required:    true,
				Description: "The bot detection plugin type",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"detection_type": schema.StringAttribute{
				Required:    true,
				Description: "The detection type",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"configuration": schema.StringAttribute{
				Required:    true,
				Description: "JSON configuration for the bot detection plugin",
			},
		},
	}
}

func (r *BotDetectionResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *BotDetectionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan BotDetectionModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := buildCreateBody(plan)

	result, err := r.client.CreateBotDetection(ctx, plan.DomainID.ValueString(), body)
	if err != nil {
		resp.Diagnostics.AddError("Error creating bot detection", err.Error())
		return
	}

	plan.ID = types.StringValue(result["id"].(string))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *BotDetectionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state BotDetectionModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.GetBotDetection(ctx, state.DomainID.ValueString(), state.ID.ValueString())
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading bot detection", err.Error())
		return
	}

	readIntoModel(&state, result)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *BotDetectionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan BotDetectionModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state BotDetectionModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.ID = state.ID

	body := buildUpdateBody(plan)

	_, err := r.client.UpdateBotDetection(ctx, plan.DomainID.ValueString(), plan.ID.ValueString(), body)
	if err != nil {
		resp.Diagnostics.AddError("Error updating bot detection", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *BotDetectionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state BotDetectionModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteBotDetection(ctx, state.DomainID.ValueString(), state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error deleting bot detection", err.Error())
	}
}

func (r *BotDetectionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	domainID, botDetectionID, ok := parseImportID(req.ID)
	if !ok {
		resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("Expected format: domain_id/bot_detection_id, got: %s", req.ID))
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("domain_id"), domainID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), botDetectionID)...)
}

func parseImportID(id string) (string, string, bool) {
	parts := strings.SplitN(id, "/", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	return parts[0], parts[1], true
}

func buildCreateBody(plan BotDetectionModel) map[string]interface{} {
	body := buildUpdateBody(plan)
	body["detectionType"] = plan.DetectionType.ValueString()
	return body
}

func buildUpdateBody(plan BotDetectionModel) map[string]interface{} {
	return map[string]interface{}{
		"name":          plan.Name.ValueString(),
		"type":          plan.Type.ValueString(),
		"configuration": plan.Configuration.ValueString(),
	}
}

func readIntoModel(model *BotDetectionModel, data map[string]interface{}) {
	if name, ok := data["name"].(string); ok {
		model.Name = types.StringValue(name)
	}
	if t, ok := data["type"].(string); ok {
		model.Type = types.StringValue(t)
	}
	if dt, ok := data["detectionType"].(string); ok {
		model.DetectionType = types.StringValue(dt)
	}
	// Configuration may contain masked secrets, so preserve it from state.
}
