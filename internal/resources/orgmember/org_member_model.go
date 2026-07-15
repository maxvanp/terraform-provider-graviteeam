package orgmember

import "github.com/hashicorp/terraform-plugin-framework/types"

type OrgMemberModel struct {
	ID         types.String `tfsdk:"id"`
	MemberID   types.String `tfsdk:"member_id"`
	MemberType types.String `tfsdk:"member_type"`
	RoleID     types.String `tfsdk:"role_id"`
}
