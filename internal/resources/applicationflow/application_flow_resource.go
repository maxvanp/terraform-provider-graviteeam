package applicationflow

import (
	"context"
	"encoding/json"
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
	_ resource.Resource                = &ApplicationFlowResource{}
	_ resource.ResourceWithImportState = &ApplicationFlowResource{}
)

type ApplicationFlowResource struct {
	client *client.Client
}

func NewApplicationFlowResource() resource.Resource {
	return &ApplicationFlowResource{}
}

func (r *ApplicationFlowResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_application_flow"
}

func (r *ApplicationFlowResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages authentication flows for a Gravitee AM application. Flows exist by default; this resource configures them via GET/PUT.",
		Attributes: map[string]schema.Attribute{
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
			"flows": schema.StringAttribute{
				Required:    true,
				Description: "JSON string containing the flow definitions. PUT replaces the entire flow list.",
			},
		},
	}
}

func (r *ApplicationFlowResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ApplicationFlowResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ApplicationFlowModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var flowsData []interface{}
	if err := json.Unmarshal([]byte(plan.Flows.ValueString()), &flowsData); err != nil {
		resp.Diagnostics.AddError("Invalid flows JSON", fmt.Sprintf("Error parsing flows: %s", err))
		return
	}

	result, err := r.client.UpdateApplicationFlows(ctx, plan.DomainID.ValueString(), plan.ApplicationID.ValueString(), flowsData)
	if err != nil {
		resp.Diagnostics.AddError("Error creating application flows", err.Error())
		return
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		resp.Diagnostics.AddError("Error marshaling flows response", err.Error())
		return
	}
	plan.Flows = types.StringValue(string(resultJSON))

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ApplicationFlowResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ApplicationFlowModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.GetApplicationFlows(ctx, state.DomainID.ValueString(), state.ApplicationID.ValueString())
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading application flows", err.Error())
		return
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		resp.Diagnostics.AddError("Error marshaling flows response", err.Error())
		return
	}
	state.Flows = types.StringValue(string(resultJSON))

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ApplicationFlowResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ApplicationFlowModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var flowsData []interface{}
	if err := json.Unmarshal([]byte(plan.Flows.ValueString()), &flowsData); err != nil {
		resp.Diagnostics.AddError("Invalid flows JSON", fmt.Sprintf("Error parsing flows: %s", err))
		return
	}

	result, err := r.client.UpdateApplicationFlows(ctx, plan.DomainID.ValueString(), plan.ApplicationID.ValueString(), flowsData)
	if err != nil {
		resp.Diagnostics.AddError("Error updating application flows", err.Error())
		return
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		resp.Diagnostics.AddError("Error marshaling flows response", err.Error())
		return
	}
	plan.Flows = types.StringValue(string(resultJSON))

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ApplicationFlowResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ApplicationFlowModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Reset flows to empty list (flows always exist, we just clear them)
	_, err := r.client.UpdateApplicationFlows(ctx, state.DomainID.ValueString(), state.ApplicationID.ValueString(), []interface{}{})
	if err != nil {
		resp.Diagnostics.AddError("Error deleting application flows", err.Error())
	}
}

func (r *ApplicationFlowResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import format: domain_id/application_id
	parts := strings.Split(req.ID, "/")
	if len(parts) != 2 {
		resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("Expected format: domain_id/application_id, got: %s", req.ID))
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("domain_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("application_id"), parts[1])...)
}
