package identityproviderpasswordpolicy

import "github.com/hashicorp/terraform-plugin-framework/types"

type IdentityProviderPasswordPolicyModel struct {
	ID                 types.String `tfsdk:"id"`
	DomainID           types.String `tfsdk:"domain_id"`
	IdentityProviderID types.String `tfsdk:"identity_provider_id"`
	PasswordPolicyID   types.String `tfsdk:"password_policy_id"`
}
