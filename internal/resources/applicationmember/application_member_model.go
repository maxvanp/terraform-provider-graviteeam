package applicationmember

import "github.com/hashicorp/terraform-plugin-framework/types"

type ApplicationMemberModel struct {
	ID            types.String `tfsdk:"id"`
	DomainID      types.String `tfsdk:"domain_id"`
	ApplicationID types.String `tfsdk:"application_id"`
	MemberID      types.String `tfsdk:"member_id"`
	MemberType    types.String `tfsdk:"member_type"`
	RoleID        types.String `tfsdk:"role_id"`
}
