package orgtag

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/maxvanp/terraform-provider-graviteeam/internal/client"
)

var (
	_ resource.Resource                = &OrgTagResource{}
	_ resource.ResourceWithImportState = &OrgTagResource{}
)

type OrgTagResource struct {
	client *client.Client
}

func NewOrgTagResource() resource.Resource {
	return &OrgTagResource{}
}

func (r *OrgTagResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_org_tag"
}

func (r *OrgTagResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Gravitee AM organization tag (sharding tag)",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the organization tag",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the organization tag",
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: "The description of the organization tag",
			},
		},
	}
}

func (r *OrgTagResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *OrgTagResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan OrgTagModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := buildBody(plan, nil)

	result, err := r.client.CreateOrgTag(ctx, body)
	if err != nil {
		resp.Diagnostics.AddError("Error creating organization tag", err.Error())
		return
	}

	plan.ID = types.StringValue(result["id"].(string))
	readIntoModel(&plan, result)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *OrgTagResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state OrgTagModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.GetOrgTag(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading organization tag", err.Error())
		return
	}

	readIntoModel(&state, result)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *OrgTagResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan OrgTagModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state OrgTagModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.ID = state.ID

	body := buildBody(plan, &state)

	_, err := r.client.UpdateOrgTag(ctx, plan.ID.ValueString(), body)
	if err != nil {
		resp.Diagnostics.AddError("Error updating organization tag", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *OrgTagResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state OrgTagModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteOrgTag(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error deleting organization tag", err.Error())
	}
}

func (r *OrgTagResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func buildBody(plan OrgTagModel, state *OrgTagModel) map[string]interface{} {
	body := map[string]interface{}{
		"name": plan.Name.ValueString(),
	}
	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		body["description"] = plan.Description.ValueString()
	} else if state != nil && !state.Description.IsNull() {
		body["description"] = ""
	}
	return body
}

func readIntoModel(model *OrgTagModel, result map[string]interface{}) {
	if name, ok := result["name"].(string); ok {
		model.Name = types.StringValue(name)
	}
	if desc, ok := result["description"].(string); ok && desc != "" {
		model.Description = types.StringValue(desc)
	} else {
		model.Description = types.StringNull()
	}
}
