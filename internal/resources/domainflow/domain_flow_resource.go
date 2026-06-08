package domainflow

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
	_ resource.Resource                = &DomainFlowResource{}
	_ resource.ResourceWithImportState = &DomainFlowResource{}
)

type DomainFlowResource struct {
	client *client.Client
}

func NewDomainFlowResource() resource.Resource {
	return &DomainFlowResource{}
}

func (r *DomainFlowResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_domain_flow"
}

func (r *DomainFlowResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages authentication flows for a Gravitee AM domain. Flows exist by default; this resource configures them via GET/PUT.",
		Attributes: map[string]schema.Attribute{
			"domain_id": schema.StringAttribute{
				Required:    true,
				Description: "The ID of the domain",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"flows": schema.StringAttribute{
				Required:    true,
				Description: "JSON string containing the domain flow definitions. PUT replaces the entire flow list.",
			},
		},
	}
}

func (r *DomainFlowResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *DomainFlowResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan DomainFlowModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	flowsData, ok := decodeFlows(plan.Flows.ValueString(), &resp.Diagnostics)
	if !ok {
		return
	}

	result, err := r.client.UpdateDomainFlows(ctx, plan.DomainID.ValueString(), flowsData)
	if err != nil {
		resp.Diagnostics.AddError("Error creating domain flows", err.Error())
		return
	}

	flowsJSON, err := encodeFlows(result)
	if err != nil {
		resp.Diagnostics.AddError("Error marshaling flows response", err.Error())
		return
	}
	plan.Flows = types.StringValue(flowsJSON)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *DomainFlowResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state DomainFlowModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.ListFlows(ctx, state.DomainID.ValueString())
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading domain flows", err.Error())
		return
	}

	flowsJSON, err := encodeFlows(result)
	if err != nil {
		resp.Diagnostics.AddError("Error marshaling flows response", err.Error())
		return
	}
	state.Flows = types.StringValue(flowsJSON)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *DomainFlowResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan DomainFlowModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	flowsData, ok := decodeFlows(plan.Flows.ValueString(), &resp.Diagnostics)
	if !ok {
		return
	}

	result, err := r.client.UpdateDomainFlows(ctx, plan.DomainID.ValueString(), flowsData)
	if err != nil {
		resp.Diagnostics.AddError("Error updating domain flows", err.Error())
		return
	}

	flowsJSON, err := encodeFlows(result)
	if err != nil {
		resp.Diagnostics.AddError("Error marshaling flows response", err.Error())
		return
	}
	plan.Flows = types.StringValue(flowsJSON)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *DomainFlowResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state DomainFlowModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.UpdateDomainFlows(ctx, state.DomainID.ValueString(), []interface{}{})
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			return
		}
		resp.Diagnostics.AddError("Error deleting domain flows", err.Error())
	}
}

func (r *DomainFlowResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("domain_id"), req.ID)...)
}

func decodeFlows(value string, diagnostics interface {
	AddError(summary string, detail string)
}) ([]interface{}, bool) {
	var flowsData []interface{}
	if err := json.Unmarshal([]byte(value), &flowsData); err != nil {
		diagnostics.AddError("Invalid flows JSON", fmt.Sprintf("Error parsing flows: %s", err))
		return nil, false
	}
	return flowsData, true
}

func encodeFlows(value interface{}) (string, error) {
	flowsJSON, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return string(flowsJSON), nil
}
