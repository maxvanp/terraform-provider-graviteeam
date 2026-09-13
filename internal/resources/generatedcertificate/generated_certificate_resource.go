package generatedcertificate

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
	_ resource.Resource                = &GeneratedCertificateResource{}
	_ resource.ResourceWithImportState = &GeneratedCertificateResource{}
)

type GeneratedCertificateResource struct {
	client *client.Client
}

func NewGeneratedCertificateResource() resource.Resource {
	return &GeneratedCertificateResource{}
}

func (r *GeneratedCertificateResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_generated_certificate"
}

func (r *GeneratedCertificateResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Gravitee AM generated system certificate created by the certificate rotation endpoint.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the generated certificate.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"domain_id": schema.StringAttribute{
				Required:    true,
				Description: "The ID of the domain.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"rotation_trigger": schema.StringAttribute{
				Optional:    true,
				Description: "Arbitrary value used to explicitly replace the generated certificate. Changing this value creates a new generated certificate and deletes the old one.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Computed:    true,
				Description: "The generated certificate name returned by Gravitee AM.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"type": schema.StringAttribute{
				Computed:    true,
				Description: "The generated certificate plugin type returned by Gravitee AM.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *GeneratedCertificateResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *GeneratedCertificateResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan GeneratedCertificateModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.RotateCertificate(ctx, plan.DomainID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error generating certificate", err.Error())
		return
	}

	id, ok := certificateID(result)
	if !ok {
		resp.Diagnostics.AddError("Error generating certificate", "API response did not include a certificate id")
		return
	}
	plan.ID = types.StringValue(id)
	readGeneratedCertificate(&plan, result)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *GeneratedCertificateResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state GeneratedCertificateModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.GetCertificate(ctx, state.DomainID.ValueString(), state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading generated certificate", err.Error())
		return
	}

	readGeneratedCertificate(&state, result)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *GeneratedCertificateResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var state GeneratedCertificateModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *GeneratedCertificateResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state GeneratedCertificateModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteCertificate(ctx, state.DomainID.ValueString(), state.ID.ValueString()); err != nil {
		if client.IsNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Error deleting generated certificate", err.Error())
	}
}

func (r *GeneratedCertificateResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 {
		resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("Expected format: domain_id/certificate_id, got: %s", req.ID))
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("domain_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
}

func readGeneratedCertificate(model *GeneratedCertificateModel, result map[string]interface{}) {
	if name, ok := result["name"].(string); ok {
		model.Name = types.StringValue(name)
	}
	if certificateType, ok := result["type"].(string); ok {
		model.Type = types.StringValue(certificateType)
	}
}

func certificateID(result map[string]interface{}) (string, bool) {
	id, ok := result["id"].(string)
	if !ok || id == "" {
		return "", false
	}
	return id, true
}
