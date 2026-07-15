package domainmember

import "github.com/hashicorp/terraform-plugin-framework/types"

type DomainMemberModel struct {
	ID         types.String `tfsdk:"id"`
	DomainID   types.String `tfsdk:"domain_id"`
	MemberID   types.String `tfsdk:"member_id"`
	MemberType types.String `tfsdk:"member_type"`
	RoleID     types.String `tfsdk:"role_id"`
}
