package domaincertificatesettings

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
	_ resource.Resource                = &DomainCertificateSettingsResource{}
	_ resource.ResourceWithImportState = &DomainCertificateSettingsResource{}
)

type DomainCertificateSettingsResource struct {
	client *client.Client
}

func NewDomainCertificateSettingsResource() resource.Resource {
	return &DomainCertificateSettingsResource{}
}

func (r *DomainCertificateSettingsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_domain_certificate_settings"
}

func (r *DomainCertificateSettingsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages Gravitee AM domain certificate settings",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The Terraform ID of the domain certificate settings",
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
			"fallback_certificate_id": schema.StringAttribute{
				Required:    true,
				Description: "The ID of the fallback certificate",
			},
		},
	}
}

func (r *DomainCertificateSettingsResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *DomainCertificateSettingsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan DomainCertificateSettingsModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.update(ctx, &plan, plan.FallbackCertificateID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error creating domain certificate settings", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *DomainCertificateSettingsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state DomainCertificateSettingsModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.GetDomain(ctx, state.DomainID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading domain certificate settings", err.Error())
		return
	}

	fallbackCertificateID := readFallbackCertificateID(result)
	if fallbackCertificateID == "" {
		resp.State.RemoveResource(ctx)
		return
	}

	state.ID = state.DomainID
	state.FallbackCertificateID = types.StringValue(fallbackCertificateID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *DomainCertificateSettingsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan DomainCertificateSettingsModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.update(ctx, &plan, plan.FallbackCertificateID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error updating domain certificate settings", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *DomainCertificateSettingsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state DomainCertificateSettingsModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.UpdateDomainCertificateSettings(ctx, state.DomainID.ValueString(), buildDeleteBody())
	if err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Error deleting domain certificate settings", err.Error())
	}
}

func (r *DomainCertificateSettingsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("domain_id"), req.ID)...)
}

func (r *DomainCertificateSettingsResource) update(ctx context.Context, model *DomainCertificateSettingsModel, fallbackCertificateID string) error {
	_, err := r.client.UpdateDomainCertificateSettings(ctx, model.DomainID.ValueString(), buildBody(fallbackCertificateID))
	if err != nil {
		return err
	}
	model.ID = model.DomainID
	return nil
}

func buildBody(fallbackCertificateID string) map[string]interface{} {
	return map[string]interface{}{
		"fallbackCertificate": fallbackCertificateID,
	}
}

func buildDeleteBody() map[string]interface{} {
	return map[string]interface{}{
		"fallbackCertificate": nil,
	}
}

func readFallbackCertificateID(domain map[string]interface{}) string {
	certificateSettings, ok := domain["certificateSettings"].(map[string]interface{})
	if !ok {
		return ""
	}
	fallbackCertificateID, _ := certificateSettings["fallbackCertificate"].(string)
	return fallbackCertificateID
}
