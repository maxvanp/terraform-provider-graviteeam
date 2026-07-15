package protectedresourcesecret

import "github.com/hashicorp/terraform-plugin-framework/types"

type ProtectedResourceSecretModel struct {
	ID                  types.String `tfsdk:"id"`
	DomainID            types.String `tfsdk:"domain_id"`
	ProtectedResourceID types.String `tfsdk:"protected_resource_id"`
	Name                types.String `tfsdk:"name"`
	RenewTrigger        types.String `tfsdk:"renew_trigger"`
	Secret              types.String `tfsdk:"secret"`
	SettingsID          types.String `tfsdk:"settings_id"`
	ExpiresAt           types.String `tfsdk:"expires_at"`
}
