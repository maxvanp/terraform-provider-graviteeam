package orgidentityprovider

import "github.com/hashicorp/terraform-plugin-framework/types"

type OrgIdentityProviderModel struct {
	ID              types.String            `tfsdk:"id"`
	Name            types.String            `tfsdk:"name"`
	Type            types.String            `tfsdk:"type"`
	Configuration   types.String            `tfsdk:"configuration"`
	Mappers         map[string]types.String `tfsdk:"mappers"`
	DomainWhitelist []types.String          `tfsdk:"domain_whitelist"`
	External        types.Bool              `tfsdk:"external"`
	GroupMapper     types.Map               `tfsdk:"group_mapper"`
	RoleMapper      types.Map               `tfsdk:"role_mapper"`
}
