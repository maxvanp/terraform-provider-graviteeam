package orgrole

import "github.com/hashicorp/terraform-plugin-framework/types"

type OrgRoleModel struct {
	ID             types.String   `tfsdk:"id"`
	Name           types.String   `tfsdk:"name"`
	Description    types.String   `tfsdk:"description"`
	AssignableType types.String   `tfsdk:"assignable_type"`
	Permissions    []types.String `tfsdk:"permissions"`
}
