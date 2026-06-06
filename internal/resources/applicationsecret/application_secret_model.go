package applicationsecret

import "github.com/hashicorp/terraform-plugin-framework/types"

type ApplicationSecretModel struct {
	ID            types.String `tfsdk:"id"`
	DomainID      types.String `tfsdk:"domain_id"`
	ApplicationID types.String `tfsdk:"application_id"`
	Name          types.String `tfsdk:"name"`
	RenewTrigger  types.String `tfsdk:"renew_trigger"`
	Secret        types.String `tfsdk:"secret"`
	SettingsID    types.String `tfsdk:"settings_id"`
	ExpiresAt     types.String `tfsdk:"expires_at"`
}
