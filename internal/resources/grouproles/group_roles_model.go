package grouproles

import "github.com/hashicorp/terraform-plugin-framework/types"

type GroupRolesModel struct {
	DomainID types.String   `tfsdk:"domain_id"`
	GroupID  types.String   `tfsdk:"group_id"`
	Roles    []types.String `tfsdk:"roles"`
}
