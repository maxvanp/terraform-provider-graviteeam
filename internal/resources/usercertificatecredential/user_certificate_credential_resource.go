package usercertificatecredential

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
	"github.com/maxvanp/terraform-provider-graviteeam/internal/resources/importid"
)

var (
	_ resource.Resource                = &UserCertificateCredentialResource{}
	_ resource.ResourceWithImportState = &UserCertificateCredentialResource{}
)

type UserCertificateCredentialResource struct {
	client *client.Client
}

func NewUserCertificateCredentialResource() resource.Resource {
	return &UserCertificateCredentialResource{}
}

func (r *UserCertificateCredentialResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user_certificate_credential"
}

func (r *UserCertificateCredentialResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Gravitee AM domain user certificate credential",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the certificate credential",
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
			"user_id": schema.StringAttribute{
				Required:    true,
				Description: "The ID of the user",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"certificate_pem": schema.StringAttribute{
				Required:    true,
				Description: "The certificate in PEM format",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"certificate_thumbprint": schema.StringAttribute{
				Computed:    true,
				Description: "The certificate thumbprint",
			},
			"certificate_subject_dn": schema.StringAttribute{
				Computed:    true,
				Description: "The certificate subject distinguished name",
			},
			"certificate_serial_number": schema.StringAttribute{
				Computed:    true,
				Description: "The certificate serial number",
			},
			"certificate_issuer_dn": schema.StringAttribute{
				Computed:    true,
				Description: "The certificate issuer distinguished name",
			},
			"certificate_expires_at": schema.StringAttribute{
				Computed:    true,
				Description: "The certificate expiration timestamp returned by Gravitee AM",
			},
			"username": schema.StringAttribute{
				Computed:    true,
				Description: "The username associated with the certificate credential",
			},
		},
	}
}

func (r *UserCertificateCredentialResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *UserCertificateCredentialResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan UserCertificateCredentialModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.CreateUserCertificateCredential(ctx, plan.DomainID.ValueString(), plan.UserID.ValueString(), buildCreateBody(plan))
	if err != nil {
		resp.Diagnostics.AddError("Error creating user certificate credential", err.Error())
		return
	}

	readIntoModel(&plan, result)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *UserCertificateCredentialResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state UserCertificateCredentialModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.GetUserCertificateCredential(ctx, state.DomainID.ValueString(), state.UserID.ValueString(), state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading user certificate credential", err.Error())
		return
	}

	readIntoModel(&state, result)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *UserCertificateCredentialResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Unsupported User Certificate Credential Update", "User certificate credential attributes require replacement.")
}

func (r *UserCertificateCredentialResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state UserCertificateCredentialModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteUserCertificateCredential(ctx, state.DomainID.ValueString(), state.UserID.ValueString(), state.ID.ValueString())
	if err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Error deleting user certificate credential", err.Error())
	}
}

func (r *UserCertificateCredentialResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, "/")
	if len(parts) != 3 || !importid.Valid(parts) {
		resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("Expected format: domain_id/user_id/credential_id, got: %s", req.ID))
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("domain_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("user_id"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[2])...)
}

func readIntoModel(model *UserCertificateCredentialModel, data map[string]interface{}) {
	if id, ok := data["id"].(string); ok {
		model.ID = types.StringValue(id)
	}
	if certificatePEM, ok := data["certificatePem"].(string); ok {
		model.CertificatePEM = types.StringValue(certificatePEM)
	}
	if thumbprint, ok := data["certificateThumbprint"].(string); ok {
		model.CertificateThumbprint = types.StringValue(thumbprint)
	} else {
		model.CertificateThumbprint = types.StringNull()
	}
	if subjectDN, ok := data["certificateSubjectDN"].(string); ok {
		model.CertificateSubjectDN = types.StringValue(subjectDN)
	} else {
		model.CertificateSubjectDN = types.StringNull()
	}
	if serialNumber, ok := data["certificateSerialNumber"].(string); ok {
		model.CertificateSerialNumber = types.StringValue(serialNumber)
	} else {
		model.CertificateSerialNumber = types.StringNull()
	}
	if issuerDN, ok := data["certificateIssuerDN"].(string); ok {
		model.CertificateIssuerDN = types.StringValue(issuerDN)
	} else {
		model.CertificateIssuerDN = types.StringNull()
	}
	model.CertificateExpiresAt = timestampString(data["certificateExpiresAt"])
	if username, ok := data["username"].(string); ok {
		model.Username = types.StringValue(username)
	} else {
		model.Username = types.StringNull()
	}
}

func timestampString(value interface{}) types.String {
	switch typed := value.(type) {
	case string:
		return types.StringValue(typed)
	case float64:
		return types.StringValue(fmt.Sprintf("%.0f", typed))
	default:
		return types.StringNull()
	}
}

func buildCreateBody(plan UserCertificateCredentialModel) map[string]interface{} {
	return map[string]interface{}{
		"certificatePem": plan.CertificatePEM.ValueString(),
	}
}
