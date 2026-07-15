package generatedcertificate

import "github.com/hashicorp/terraform-plugin-framework/types"

type GeneratedCertificateModel struct {
	ID              types.String `tfsdk:"id"`
	DomainID        types.String `tfsdk:"domain_id"`
	RotationTrigger types.String `tfsdk:"rotation_trigger"`
	Name            types.String `tfsdk:"name"`
	Type            types.String `tfsdk:"type"`
}
