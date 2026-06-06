package protectedresourcemember

import "github.com/hashicorp/terraform-plugin-framework/types"

type ProtectedResourceMemberModel struct {
	ID                  types.String `tfsdk:"id"`
	DomainID            types.String `tfsdk:"domain_id"`
	ProtectedResourceID types.String `tfsdk:"protected_resource_id"`
	MemberID            types.String `tfsdk:"member_id"`
	MemberType          types.String `tfsdk:"member_type"`
	RoleID              types.String `tfsdk:"role_id"`
}
