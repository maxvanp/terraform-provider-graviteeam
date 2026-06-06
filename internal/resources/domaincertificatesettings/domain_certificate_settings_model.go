package domaincertificatesettings

import "github.com/hashicorp/terraform-plugin-framework/types"

type DomainCertificateSettingsModel struct {
	ID                    types.String `tfsdk:"id"`
	DomainID              types.String `tfsdk:"domain_id"`
	FallbackCertificateID types.String `tfsdk:"fallback_certificate_id"`
}
