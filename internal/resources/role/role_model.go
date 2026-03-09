package role

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type RoleModel struct {
	ID             types.String   `tfsdk:"id"`
	DomainID       types.String   `tfsdk:"domain_id"`
	Name           types.String   `tfsdk:"name"`
	Description    types.String   `tfsdk:"description"`
	AssignableType types.String   `tfsdk:"assignable_type"`
	Permissions    []types.String `tfsdk:"permissions"`
	OAuthScopes    []types.String `tfsdk:"oauth_scopes"`
}
