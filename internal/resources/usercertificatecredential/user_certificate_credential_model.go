package usercertificatecredential

import "github.com/hashicorp/terraform-plugin-framework/types"

type UserCertificateCredentialModel struct {
	ID                      types.String `tfsdk:"id"`
	DomainID                types.String `tfsdk:"domain_id"`
	UserID                  types.String `tfsdk:"user_id"`
	CertificatePEM          types.String `tfsdk:"certificate_pem"`
	CertificateThumbprint   types.String `tfsdk:"certificate_thumbprint"`
	CertificateSubjectDN    types.String `tfsdk:"certificate_subject_dn"`
	CertificateSerialNumber types.String `tfsdk:"certificate_serial_number"`
	CertificateIssuerDN     types.String `tfsdk:"certificate_issuer_dn"`
	CertificateExpiresAt    types.String `tfsdk:"certificate_expires_at"`
	Username                types.String `tfsdk:"username"`
}
