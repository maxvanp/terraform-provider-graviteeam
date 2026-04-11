package userrole

import "github.com/hashicorp/terraform-plugin-framework/types"

type UserRoleModel struct {
	DomainID types.String   `tfsdk:"domain_id"`
	UserID   types.String   `tfsdk:"user_id"`
	Roles    []types.String `tfsdk:"roles"`
}
