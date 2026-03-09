package identityprovider

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type IdentityProviderModel struct {
	ID               types.String            `tfsdk:"id"`
	DomainID         types.String            `tfsdk:"domain_id"`
	Name             types.String            `tfsdk:"name"`
	Type             types.String            `tfsdk:"type"`
	External         types.Bool              `tfsdk:"external"`
	Configuration    types.String            `tfsdk:"configuration"`
	Mappers          map[string]types.String `tfsdk:"mappers"`
	DomainWhitelist  []types.String          `tfsdk:"domain_whitelist"`
	PasswordPolicyID types.String            `tfsdk:"password_policy_id"`
	RoleMapper       types.Map               `tfsdk:"role_mapper"`
}
